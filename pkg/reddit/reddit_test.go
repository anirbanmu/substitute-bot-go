package reddit

import (
	"bytes"
	"encoding/json"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/ugorji/go/codec"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

var _ = Describe("Reddit", func() {
	var server *ghttp.Server
	var client *http.Client
	var api *Api

	creds := Credentials{
		"dummy-username",
		"dummy-password",
		"dummy-clientid",
		"dummy-clientsecret",
		"dummy-user-agent",
	}
	token := "dummy-access-token"
	encodeBuffer := bytes.Buffer{}

	BeforeEach(func() {
		server = ghttp.NewServer()

		serverURL, err := url.Parse(server.URL())
		Expect(err).ToNot(HaveOccurred())

		dialMock := func(network, addr string) (net.Conn, error) {
			return net.Dial(network, serverURL.Host)
		}

		client = &http.Client{
			Transport: &http.Transport{
				Dial:    dialMock,
				DialTLS: dialMock,
			},
		}

		api = &Api{
			creds:        creds,
			Client:       client,
			token:        token,
			grantTime:    time.Now(),
			EncodeBuffer: &encodeBuffer,
			Encoder:      codec.NewEncoder(&encodeBuffer, &codec.JsonHandle{}),
			Decoder:      codec.NewDecoderBytes(nil, &codec.JsonHandle{}),
		}
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("InitApiFromEnv", func() {
		BeforeEach(func() {
			os.Setenv("SUBSTITUTE_BOT_USERNAME", "something")
			os.Setenv("SUBSTITUTE_BOT_PASSWORD", "something")
			os.Setenv("SUBSTITUTE_BOT_CLIENT_ID", "something")
			os.Setenv("SUBSTITUTE_BOT_CLIENT_SECRET", "something")
			os.Setenv("SUBSTITUTE_BOT_USER_AGENT", "something")
		})

		AfterEach(func() {
			os.Unsetenv("SUBSTITUTE_BOT_USERNAME")
			os.Unsetenv("SUBSTITUTE_BOT_PASSWORD")
			os.Unsetenv("SUBSTITUTE_BOT_CLIENT_ID")
			os.Unsetenv("SUBSTITUTE_BOT_CLIENT_SECRET")
			os.Unsetenv("SUBSTITUTE_BOT_USER_AGENT")
		})

		It("returns nil & error when SUBSTITUTE_BOT_USERNAME is not set", func() {
			os.Unsetenv("SUBSTITUTE_BOT_USERNAME")
			createdApi, err := InitApiFromEnv(client)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("SUBSTITUTE_BOT_USERNAME"))
			Expect(createdApi).To(BeNil())
		})

		It("returns nil & error when SUBSTITUTE_BOT_PASSWORD is not set", func() {
			os.Unsetenv("SUBSTITUTE_BOT_PASSWORD")
			createdApi, err := InitApiFromEnv(client)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("SUBSTITUTE_BOT_PASSWORD"))
			Expect(createdApi).To(BeNil())
		})

		It("returns nil & error when SUBSTITUTE_BOT_CLIENT_ID is not set", func() {
			os.Unsetenv("SUBSTITUTE_BOT_CLIENT_ID")
			createdApi, err := InitApiFromEnv(client)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("SUBSTITUTE_BOT_CLIENT_ID"))
			Expect(createdApi).To(BeNil())
		})

		It("returns nil & error when SUBSTITUTE_BOT_CLIENT_SECRET is not set", func() {
			os.Unsetenv("SUBSTITUTE_BOT_CLIENT_SECRET")
			createdApi, err := InitApiFromEnv(client)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("SUBSTITUTE_BOT_CLIENT_SECRET"))
			Expect(createdApi).To(BeNil())
		})

		It("returns nil & error when SUBSTITUTE_BOT_USER_AGENT is not set", func() {
			os.Unsetenv("SUBSTITUTE_BOT_USER_AGENT")
			createdApi, err := InitApiFromEnv(client)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("SUBSTITUTE_BOT_USER_AGENT"))
			Expect(createdApi).To(BeNil())
		})
	})

	Describe("InitApi", func() {
		verificationHandlers := []http.HandlerFunc{
			ghttp.VerifyRequest("POST", "/api/v1/access_token"),
			ghttp.VerifyForm(
				url.Values{
					"grant_type": {"password"},
					"username":   {creds.Username},
					"password":   {creds.Password},
				},
			),
			ghttp.VerifyBasicAuth(creds.ClientId, creds.ClientSecret),
			ghttp.VerifyHeader(http.Header{"User-Agent": []string{creds.UserAgent}}),
		}

		Context("when all goes correctly", func() {
			It("posts the correct parameters, returns no error & an authed Api", func() {
				handlers := append(
					verificationHandlers,
					ghttp.RespondWith(http.StatusOK, "{\"access_token\":\""+token+"\"}"),
				)
				server.AppendHandlers(ghttp.CombineHandlers(handlers...))

				createdApi, err := InitApi(creds, client)
				Expect(err).NotTo(HaveOccurred())
				Expect(createdApi).NotTo(BeNil())
				Expect(createdApi.token).To(Equal(token))
			})
		})

		Context("when API returns non 200 status code", func() {
			It("returns error & no Api", func() {
				handlers := append(
					verificationHandlers,
					ghttp.RespondWith(http.StatusInternalServerError, ""),
				)
				server.AppendHandlers(ghttp.CombineHandlers(handlers...))

				createdApi, err := InitApi(creds, client)
				Expect(err).To(HaveOccurred())
				Expect(createdApi).To(BeNil())
			})
		})

		Context("when API returns 200 but no body", func() {
			It("returns error & no Api", func() {
				handlers := append(
					verificationHandlers,
					ghttp.RespondWith(http.StatusOK, ""),
				)
				server.AppendHandlers(ghttp.CombineHandlers(handlers...))

				createdApi, err := InitApi(creds, client)
				Expect(err).To(HaveOccurred())
				Expect(createdApi).To(BeNil())
			})
		})

		Context("when API returns 200 but blank access_token", func() {
			It("returns error & no Api", func() {
				handlers := append(
					verificationHandlers,
					ghttp.RespondWith(http.StatusOK, "{\"access_token\":\"\"}"),
				)
				server.AppendHandlers(ghttp.CombineHandlers(handlers...))

				createdApi, err := InitApi(creds, client)
				Expect(err).To(HaveOccurred())
				Expect(createdApi).To(BeNil())
			})
		})

		Context("when there is a network error", func() {
			BeforeEach(func() {
				dialMock := func(network, addr string) (net.Conn, error) {
					return net.Dial(network, "")
				}

				client = &http.Client{
					Transport: &http.Transport{
						Dial:    dialMock,
						DialTLS: dialMock,
					},
				}
			})

			It("returns error & no Api", func() {
				createdApi, err := InitApi(creds, client)
				Expect(err).To(HaveOccurred())
				Expect(createdApi).To(BeNil())
			})
		})
	})

	Describe("IsFullnameComment", func() {
		Context("when fullname begins with t1_", func() {
			It("returns true", func() {
				Expect(IsFullnameComment("t1_f3ea85d")).To(BeTrue())
			})
		})

		Context("when fullname does not begin with t1_", func() {
			It("returns false", func() {
				Expect(IsFullnameComment("t3_f3ea85d")).To(BeFalse())
			})
		})

		Context("when fullname is empty", func() {
			It("returns false", func() {
				Expect(IsFullnameComment("")).To(BeFalse())
			})
		})
	})

	comment := Comment{
		Author:         "dummy-author",
		AuthorFullname: "t2_4jtui7g8",
		Body:           "this is fake text",
		BodyHtml:       "<div class=\"md\"><p>this is fake text</p></div>",
		CreatedUtc:     1571002615,
		Id:             "g7krui4",
		Name:           "t1_g7krui4",
		ParentId:       "t1_h7kxui2",
		Permalink:      "/r/dummy-subreddit/comments/krtjrk/dummy-topic/g7krui4/",
	}
	commentJson, _ := json.Marshal(&comment)

	Describe("GetComment", func() {
		var verificationHandlers []http.HandlerFunc

		BeforeEach(func() {
			verificationHandlers = []http.HandlerFunc{
				ghttp.VerifyRequest("GET", "/api/info", "id="+comment.Name+"&raw_json=1"),
				ghttp.VerifyHeader(http.Header{
					"User-Agent":    []string{creds.UserAgent},
					"Authorization": []string{"bearer " + api.token},
				}),
			}
		})

		Context("when all goes correctly", func() {
			commentInfoJson := `{"kind":"Listing","data":{"modhash":null,"dist":1,"children":[{"kind":"t1","data":` + string(commentJson) + `}],"after":null,"before":null}}`
			It("returns no error & Comment", func() {
				handlers := append(
					verificationHandlers,
					ghttp.RespondWith(http.StatusOK, commentInfoJson),
				)
				server.AppendHandlers(ghttp.CombineHandlers(handlers...))

				c, err := api.GetComment(comment.Name)
				Expect(err).NotTo(HaveOccurred())
				Expect(c).NotTo(BeNil())
				Expect(*c).To(Equal(comment))
			})
		})

		Context("when info API returns success, but no results are given", func() {
			It("returns error & no Comment", func() {
				handlers := append(
					verificationHandlers,
					ghttp.RespondWith(http.StatusOK, `{"kind":"Listing","data":{"modhash":null,"dist":0,"children":[],"after":null,"before":null}}`),
				)
				server.AppendHandlers(ghttp.CombineHandlers(handlers...))

				c, err := api.GetComment(comment.Name)
				Expect(err).To(HaveOccurred())
				Expect(c).To(BeNil())
			})
		})

		Context("when API returns non 200 status code", func() {
			It("returns error & no Comment", func() {
				handlers := append(
					verificationHandlers,
					ghttp.RespondWith(http.StatusInternalServerError, ""),
				)
				server.AppendHandlers(ghttp.CombineHandlers(handlers...))

				c, err := api.GetComment(comment.Name)
				Expect(err).To(HaveOccurred())
				Expect(c).To(BeNil())
			})
		})

		Context("when there is a network error", func() {
			BeforeEach(func() {
				dialMock := func(network, addr string) (net.Conn, error) {
					return net.Dial(network, "")
				}

				client = &http.Client{
					Transport: &http.Transport{
						Dial:    dialMock,
						DialTLS: dialMock,
					},
				}

				api.Client = client
			})

			It("returns error & no Comment", func() {
				c, err := api.GetComment(comment.Name)
				Expect(err).To(HaveOccurred())
				Expect(c).To(BeNil())
			})
		})

		Context("when a non-comment full name is given", func() {
			It("returns error & no comment", func() {
				c, err := api.GetComment("t3_f3ea85d")
				Expect(err).To(HaveOccurred())
				Expect(c).To(BeNil())
			})
		})
	})

	Describe("PostComment", func() {
		parentId := "t1_h7kxui2"
		bodyMd := "**some** markdown"

		var verificationHandlers []http.HandlerFunc

		BeforeEach(func() {
			verificationHandlers = []http.HandlerFunc{
				ghttp.VerifyRequest("POST", "/api/comment"),
				ghttp.VerifyHeader(http.Header{
					"User-Agent":    []string{creds.UserAgent},
					"Authorization": []string{"bearer " + api.token},
				}),
				ghttp.VerifyForm(
					url.Values{
						"raw_json": {"1"},
						"api_type": {"json"},
						"thing_id": {parentId},
						"text":     {bodyMd},
					},
				),
			}
		})

		Context("when all goes correctly", func() {
			commentInfoJson := `{"json":{"errors":[],"data":{"things":[{"kind":"t1","data":` + string(commentJson) + `}]}}}`

			It("returns no error & posted Comment", func() {
				handlers := append(
					verificationHandlers,
					ghttp.RespondWith(http.StatusOK, commentInfoJson),
				)
				server.AppendHandlers(ghttp.CombineHandlers(handlers...))

				c, err := api.PostComment(parentId, bodyMd)
				Expect(err).NotTo(HaveOccurred())
				Expect(c).NotTo(BeNil())
				Expect(*c).To(Equal(comment))
			})
		})

		Context("when API returns success, but no results are given", func() {
			It("returns error & no Comment", func() {
				handlers := append(
					verificationHandlers,
					ghttp.RespondWith(http.StatusOK, `{"json":{"errors":[],"data":{"things":[]}}}`),
				)
				server.AppendHandlers(ghttp.CombineHandlers(handlers...))

				c, err := api.PostComment(parentId, bodyMd)
				Expect(err).To(HaveOccurred())
				Expect(c).To(BeNil())
			})
		})

		Context("when API returns non 200 status code", func() {
			It("returns error & no Comment", func() {
				handlers := append(
					verificationHandlers,
					ghttp.RespondWith(http.StatusInternalServerError, ""),
				)
				server.AppendHandlers(ghttp.CombineHandlers(handlers...))

				c, err := api.PostComment(parentId, bodyMd)
				Expect(err).To(HaveOccurred())
				Expect(c).To(BeNil())
			})
		})

		Context("when there is a network error", func() {
			BeforeEach(func() {
				dialMock := func(network, addr string) (net.Conn, error) {
					return net.Dial(network, "")
				}

				client = &http.Client{
					Transport: &http.Transport{
						Dial:    dialMock,
						DialTLS: dialMock,
					},
				}

				api.Client = client
			})

			It("returns error & no Comment", func() {
				c, err := api.PostComment(parentId, bodyMd)
				Expect(err).To(HaveOccurred())
				Expect(c).To(BeNil())
			})
		})

		Context("when API returns 200 but errors are present", func() {
			It("returns error & no Comment", func() {
				handlers := append(
					verificationHandlers,
					ghttp.RespondWith(http.StatusOK, `{"json":{"errors":[["NO_TEXT","we need something here","text"]],"data":{"things":[]}}}`),
				)
				server.AppendHandlers(ghttp.CombineHandlers(handlers...))

				c, err := api.PostComment(parentId, bodyMd)
				Expect(err).To(HaveOccurred())
				Expect(c).To(BeNil())
			})
		})

		Context("when no parent id is given", func() {
			It("returns error & no comment", func() {
				c, err := api.PostComment("", bodyMd)
				Expect(err).To(HaveOccurred())
				Expect(c).To(BeNil())
			})
		})

		Context("when no body is given", func() {
			It("returns error & no comment", func() {
				c, err := api.PostComment(parentId, "")
				Expect(err).To(HaveOccurred())
				Expect(c).To(BeNil())
			})
		})
	})

	Describe("reAuth", func() {
		verificationHandlers := []http.HandlerFunc{
			ghttp.VerifyRequest("POST", "/api/v1/access_token"),
			ghttp.VerifyForm(
				url.Values{
					"grant_type": {"password"},
					"username":   {creds.Username},
					"password":   {creds.Password},
				},
			),
			ghttp.VerifyBasicAuth(creds.ClientId, creds.ClientSecret),
			ghttp.VerifyHeader(http.Header{"User-Agent": []string{creds.UserAgent}}),
		}

		Context("when renewal time has not elapsed", func() {
			It("does not try to re-auth", func() {
				originalToken := api.token
				Expect(api.reAuth()).To(BeNil())
				Expect(originalToken).To(Equal(api.token))
			})
		})

		Context("when renewal time has elapsed", func() {
			BeforeEach(func() {
				api.grantTime = time.Now().Add(-1 * time.Hour)
			})

			Context("when there is an error re-authing", func() {
				It("returns error", func() {
					handlers := append(
						verificationHandlers,
						ghttp.RespondWith(http.StatusInternalServerError, ""),
					)
					server.AppendHandlers(ghttp.CombineHandlers(handlers...))

					originalToken := api.token
					originalgrantTime := api.grantTime
					Expect(api.reAuth()).To(HaveOccurred())
					Expect(api.token).To(Equal(originalToken))
					Expect(api.grantTime).To(Equal(originalgrantTime))
				})
			})

			Context("when re-auth is successful", func() {
				It("re-auth's & retrieves new token", func() {
					newToken := "new-dummy-token"
					handlers := append(
						verificationHandlers,
						ghttp.RespondWith(http.StatusOK, "{\"access_token\":\""+newToken+"\"}"),
					)
					server.AppendHandlers(ghttp.CombineHandlers(handlers...))

					originalgrantTime := api.grantTime
					Expect(api.reAuth()).NotTo(HaveOccurred())
					Expect(api.token).To(Equal(newToken))
					Expect(api.grantTime).NotTo(Equal(originalgrantTime))
				})
			})
		})
	})
})
