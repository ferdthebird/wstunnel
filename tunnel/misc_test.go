// Copyright (c) 2015 RightScale, Inc. - see LICENSE

package tunnel

// Omega: Alt+937

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Testing misc requests", func() {

	var server *ghttp.Server
	var listener net.Listener
	var wstunsrv *WSTunnelServer
	var wstuncli *WSTunnelClient
	var wstunUrl string
	var wstunToken string

	BeforeEach(func() {
		wstunToken = "test567890123456-" + strconv.Itoa(rand.Int()%1000000)
		server = ghttp.NewServer()
		fmt.Fprintf(os.Stderr, "ghttp started on %s\n", server.URL())

		listener, _ = net.Listen("tcp", "127.0.0.1:0")
		wstunsrv = NewWSTunnelServer([]string{})
		wstunsrv.Start(listener)
		fmt.Fprintf(os.Stderr, "Server started\n")
		wstuncli = NewWSTunnelClient([]string{
			"-token", wstunToken,
			"-tunnel", "ws://" + listener.Addr().String(),
			"-server", server.URL(),
			"-timeout", "3",
		})
		wstuncli.Start()
		wstunUrl = "http://" + listener.Addr().String()
	})
	AfterEach(func() {
		wstuncli.Stop()
		wstunsrv.Stop()
		server.Close()
	})

	// Perform the test by running main() with the command line args set
	It("Errors non-existing tunnels", func() {
		resp, err := http.Get(wstunUrl + "/_token/badtokenbadtoken/hello")
		Ω(err).ShouldNot(HaveOccurred())
		respBody, err := ioutil.ReadAll(resp.Body)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(string(respBody)).Should(ContainSubstring("long time"))
		Ω(resp.Header.Get("Content-Type")).Should(ContainSubstring("text/plain"))
		Ω(resp.StatusCode).Should(Equal(404))
	})

	It("Reconnects the websocket", func() {
		server.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/hello"),
				ghttp.RespondWith(200, `WORLD`,
					http.Header{"Content-Type": []string{"text/world"}}),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/hello"),
				ghttp.RespondWith(200, `AGAIN`,
					http.Header{"Content-Type": []string{"text/world"}}),
			),
		)

		// first request
		resp, err := http.Get(wstunUrl + "/_token/" + wstunToken + "/hello")
		Ω(err).ShouldNot(HaveOccurred())
		respBody, err := ioutil.ReadAll(resp.Body)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(string(respBody)).Should(Equal("WORLD"))
		Ω(resp.Header.Get("Content-Type")).Should(Equal("text/world"))
		Ω(resp.StatusCode).Should(Equal(200))

		// break the tunnel
		wstuncli.ws.Close()
		time.Sleep(20 * time.Millisecond)

		// second request
		resp, err = http.Get(wstunUrl + "/_token/" + wstunToken + "/hello")
		Ω(err).ShouldNot(HaveOccurred())
		respBody, err = ioutil.ReadAll(resp.Body)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(string(respBody)).Should(Equal("AGAIN"))
		Ω(resp.Header.Get("Content-Type")).Should(Equal("text/world"))
		Ω(resp.StatusCode).Should(Equal(200))
	})

})
