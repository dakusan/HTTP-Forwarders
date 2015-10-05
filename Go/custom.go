package main

import (
	"net/http"
//	"strings"
)

//Called before any other "http" calls
func InitServer(afh *ArgumentsForwardHandler) {

//Example (Call a custom handler for the url /custom/*)
//http.HandleFunc("/custom/", customHandler)

}

//Example custom handler
func customHandler(respWriter http.ResponseWriter, req *http.Request) {

//Example (Return some custom text)
respWriter.Write([]byte("Foobar: "+req.URL.String()))

}

//Called after request object and headers are created, but before the http request
func ModifyForwarderRequest(afh *ArgumentsForwardHandler, req *http.Request) {

//Example (add a cookie)
//req.AddCookie(&http.Cookie{Name:"foobar", Value:"baz+bar%3Afoo1"})

}

//Called after the response headers+cookies have been created and the response content host-swapped, but before the header and contents are written back
func ModifyForwarderReply(afh *ArgumentsForwardHandler, respWriter http.ResponseWriter, retContent []byte, req *http.Request) []byte {

//Example (Set a cookie)
//http.SetCookie(respWriter, &http.Cookie{Name:"foobar", Value:"baz+bar%3Afoo2"})

//Example (Add example text to the end of the html body)
//if FirstStringStartsWith(respWriter.Header()["Content-Type"], "text/", true) {
//	retContent = []byte(strings.Replace(string(retContent), "</body>", "<br>Extra text at bottom!!</body>", -1))
//}

	return retContent
}