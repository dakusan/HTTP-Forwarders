package main

import (
	"./originTypes"
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"flag"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

//Handle http requests
type ArgumentsForwardHandler struct {
	LocalPort, RemotePort                                   int
	LocalHost, RemoteHost, RemoteProtocol                   string
	IsRemoteProtocolDefaultPort, IsLocalProtocolDefaultPort string //"1"=true, ""=false - I did them as strings to counteract go’s annoying lack of the conditional operators)
	AccessOrigin                                            []originTypes.OriginType
	CustomAccessOrigin                                      string
}

func (afh *ArgumentsForwardHandler) forwardHandlerWrapper(respWriter http.ResponseWriter, req *http.Request) {
	err := afh.forwardHandler(respWriter, req)
	if err != nil {
		println(err.Error())
		http.Error(respWriter, "Bad Gateway: "+err.Error(), 502)
	}
}
func (afh *ArgumentsForwardHandler) forwardHandler(respWriter http.ResponseWriter, req *http.Request) error {
	//Get local host/port if not already determined
	if afh.LocalHost == "" {
		afh.LocalHost = strings.Split(req.Host, ":")[0]
	}

	//Update the request object
	req.Host = afh.swapHost(req.Host, false)
	req.URL.Host = req.Host
	req.URL.Scheme = afh.RemoteProtocol
	req.RequestURI = "" //go requires that this be empty
	req.RemoteAddr = "" //go likes filling this one in itself

	//Save the origin
	var origOrigin string
	hasOrigOrigin := false
	if tempVal, tempOK := req.Header["Origin"]; tempOK && len(tempVal)>0 {
		origOrigin = tempVal[0]
		hasOrigOrigin = true
	}

	//---Update the headers---
	//Swap hosts in the request headers
	for k, v := range req.Header {
		req.Header[k] = afh.swapHostArr(v, false)
		//TODO: Do cookies need to be treated separately hear?
	}
	//Update the allowed encodings
	if req.Header["Accept-Encoding"] != nil && len(req.Header["Accept-Encoding"]) > 0 {
		possibleEncodings := strings.Split(req.Header["Accept-Encoding"][0], ",")
		finalEncodings := make([]string, 0, len(possibleEncodings))
		for _, v := range possibleEncodings {
			v = strings.TrimSpace(v)
			if strings.EqualFold(v, "gzip") || strings.EqualFold(v, "deflate") {
				finalEncodings = append(finalEncodings, v)
			}
		}
		req.Header["Accept-Encoding"][0] = strings.Join(finalEncodings, ", ")
	}
	ModifyForwarderRequest(afh, req) //Custom callback

	//TODO: Should the request itself be host-swapped?

	//Send the request
	transport := http.Transport{} //Had to use a transport instead of http.Client so that Location header requests would come through instead of being followed
	resp, err := transport.RoundTrip(req)
	if err != nil {
		return err
	}

	//Save the access control origin
	var origAccessControlOrigin string
	hasOrigAccessControlOrigin := false
	if tempVal, tempOK := resp.Header["Access-Control-Allow-Origin"]; tempOK && len(tempVal)>0 {
		origAccessControlOrigin = tempVal[0]
		hasOrigAccessControlOrigin = true
	}

	//Copy the host-swapped updated return headers
	//TODO: I am not sure if this is safe right now. Some headers might need to be ignored.
	headers := respWriter.Header()
	for k, v := range resp.Header {
		if k != "Set-Cookie" { //Handle cookies separately, since they are encoded
			headers[k] = afh.swapHostArr(v, true)
		}
	}

	//Set the access control allow origin
	newOriginFound := true
	newAccessControlOrigin := func() string {
		//Test all the origin types until one succeeds
		for _, originType := range afh.AccessOrigin {
			switch originType {
				case originTypes.Original       : if hasOrigAccessControlOrigin { return origAccessControlOrigin }
				case originTypes.OriginalSwapped: if hasOrigAccessControlOrigin { return headers["Access-Control-Allow-Origin"][0] }
				case originTypes.LocalHost      : return afh.swapHost(afh.RemoteHost, true)
				case originTypes.RemoteHost     : return afh.swapHost(afh.LocalHost, false)
				case originTypes.RequestOrigin  : if hasOrigOrigin { return origOrigin }
				case originTypes.AcceptAll      : return "*"
				case originTypes.Custom         : return afh.CustomAccessOrigin
				default                         : panic("Should not get here")
			}
		}

		//If no origin was found
		newOriginFound = false
		return ""
	}()
	if !newOriginFound && hasOrigAccessControlOrigin {
		delete(headers, "Access-Control-Allow-Origin")
	} else if newOriginFound {
		headers["Access-Control-Allow-Origin"] = []string{newAccessControlOrigin}
	}

	//Copy the host-swapped returned cookies
	for _, c := range resp.Cookies() {
		unescaped, err := url.QueryUnescape(c.Value)
		if err == nil {
			c.Value = url.QueryEscape(afh.swapHost(unescaped, true))
			//TODO: More might need to be swapped here, like the Cookie.Domain
		}
		http.SetCookie(respWriter, c)
	}

	//Get the response content
	retContent, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	//Host-swap the response (only for text types)
	recompressType := ""
	if FirstStringStartsWith(resp.Header["Content-Type"], "text/", true) {
		//If compressed, decompress
		var err error
		if FirstStringStartsWith(resp.Header["Content-Encoding"], "gzip", false) {
			retContent, recompressType, err = decompress(retContent, "gzip")
		} else if FirstStringStartsWith(resp.Header["Content-Encoding"], "deflate", false) {
			retContent, recompressType, err = decompress(retContent, "deflate")
		}
		if err != nil {
			return err
		}

		//Do the swap
		retContent = []byte(afh.swapHost(string(retContent), true))
	}
	retContent = ModifyForwarderReply(afh, respWriter, retContent, req) //Custom callback

	//Recompress if required
	if recompressType != "" {
		var b bytes.Buffer
		var writer io.WriteCloser
		if recompressType == "gzip" {
			writer = gzip.NewWriter(&b)
		} else {
			writer = zlib.NewWriter(&b)
		}
		writer.Write(retContent)
		writer.Close()
		retContent = b.Bytes()
	}

	//Update the content length and write the updated headers
	headers["Content-Length"] = []string{strconv.Itoa(len(retContent))}
	respWriter.WriteHeader(resp.StatusCode) //Write the status code

	//Write the content back out
	respWriter.Write(retContent)
	return nil
}

//Decompress/deencode a stream
func decompress(input []byte, compressType string) ([]byte, string, error) {
	var newReader io.ReadCloser
	var err error
	inputByteBuffer := bytes.NewBuffer(input)
	if compressType == "gzip" {
		newReader, err = gzip.NewReader(inputByteBuffer)
	} else {
		newReader, err = zlib.NewReader(inputByteBuffer)
	}
	if err != nil {
		return input, compressType, err
	}
	returnContent, err := ioutil.ReadAll(newReader)
	newReader.Close()
	if err != nil {
		return input, compressType, err
	}
	return returnContent, compressType, nil
}

//Swap host names on request/receive strings
func (afh *ArgumentsForwardHandler) swapHostArr(strArr []string, remoteToLocal bool) []string {
	for i, v := range strArr {
		strArr[i] = afh.swapHost(v, remoteToLocal)
	}
	return strArr
}
func (afh *ArgumentsForwardHandler) swapHost(str string, remoteToLocal bool) string {
	//Determine the strings to replace and in which direction to replace them
	remoteStrings := []string{afh.RemoteHost, strconv.Itoa(afh.RemotePort), afh.IsRemoteProtocolDefaultPort}
	localStrings  := []string{afh.LocalHost, strconv.Itoa(afh.LocalPort), afh.IsLocalProtocolDefaultPort}
	findStrings, replaceStrings := remoteStrings, localStrings
	if !remoteToLocal {
		findStrings, replaceStrings = localStrings, remoteStrings
	}

	//Get the permanent replaceString, which only contains the port if not the proper http/https port
	replaceString := replaceStrings[0]
	if len(replaceStrings[2]) == 0 {
		replaceString += ":"+replaceStrings[1]
	}

	//Replace the sending host strings (HOST:PORT & HOST) with var.replaceString
	str = strings.Replace(str, findStrings[0]+":"+findStrings[1], replaceString, -1)
	str = strings.Replace(str, findStrings[0], replaceString, -1)

	return str
}

//Parse arguments and start the server
func main() {
	println("") //Always start with a blank line

	//Parse the arguments
	arguments := &ArgumentsForwardHandler{}
	const defAO string = "OriginalSwapped,RequestOrigin,LocalHost"
	var accessOrigin string
	flag.IntVar(   &arguments.LocalPort     , "LocalPort"     , 8080  , "(Optional) The local port you connect to to forward a request")
	flag.StringVar(&arguments.RemoteHost    , "RemoteHost"    , ""    , "(Required) The domain host that requests are forwarded to")
	flag.IntVar(   &arguments.RemotePort    , "RemotePort"    , 80    , "(Optional) The port for the remote domain")
	flag.StringVar(&arguments.RemoteProtocol, "RemoteProtocol", "http", "(Optional) The protocol (http/https) for the remote domain")
	flag.StringVar(&arguments.LocalHost     , "LocalHost"     , ""    , "(Optional) The host domain you are connecting to locally. In the response body, the RemoteHost is replaced by the LocalHost. The default is the domain pulled from your first http request")
	flag.StringVar(&          accessOrigin  , "AccessOrigin"  , defAO , "(Optional) How the “Access-Control-Allow-Origin” is determined. Pass “help” to get more information")
	flag.Parse()

	//Confirm the arguments
	var paramErrors []string
	if arguments.RemoteHost == "" {
		paramErrors = append(paramErrors, "Error: RemoteHost is required")
	}
	if arguments.LocalPort<1 || arguments.LocalPort>65535 {
		paramErrors = append(paramErrors, "Error: LocalPort must be between 1 and 65535")
	}
	if arguments.RemotePort<1 || arguments.RemotePort>65535 {
		paramErrors = append(paramErrors, "Error: RemotePort must be between 1 and 65535")
	}
	if arguments.RemoteProtocol!="http" && arguments.RemoteProtocol!="https" {
		paramErrors = append(paramErrors, "Error: RemoteProtocol must be “http” or “https”")
	}

	//Sets if the remote protocol is the default port
	arguments.IsRemoteProtocolDefaultPort = ""
	if (
		(arguments.RemoteProtocol == "http"  && arguments.RemotePort == 80 ) ||
		(arguments.RemoteProtocol == "https" && arguments.RemotePort == 443)) {
			arguments.IsRemoteProtocolDefaultPort = "1"
	}

	//Sets if the local protocol is the default port
	//TODO: There is not a good way to do this at the moment, so it just sets as true if the port is either http/80 or https/443
	arguments.IsLocalProtocolDefaultPort = ""
	if arguments.LocalPort == 80 || arguments.LocalPort == 443 {
		arguments.IsLocalProtocolDefaultPort = "1"
	}

	//Output information about the access origin (I like this to happen before errors cause the parameter help screen to show)
	accessOrigin = strings.ToLower(accessOrigin) //Case insensitive
	if accessOrigin == "help" {
		originTypes.GetAccessOriginHelp(true)
		return
	}

	//Parse the access origin parameter
	if accessOrigin != "" {
		accessOriginStrings := strings.Split(accessOrigin, ",")
		for i, originTypeStr := range accessOriginStrings {
			//Get and confirm the Origin Type
			originTypeStr = strings.TrimSpace(originTypeStr)
			originType, isValidOriginType := originTypes.Mapping[originTypeStr]
			if !isValidOriginType {
				paramErrors = append(paramErrors, "Invalid access origin type: "+originTypeStr)
				continue
			} else if originType==originTypes.Custom && arguments.CustomAccessOrigin!="" {
				paramErrors = append(paramErrors, "You can only have 1 “Custom” Origin Type")
				continue
			}

			//Add to the list and only
			arguments.AccessOrigin = append(arguments.AccessOrigin, originType)

			//Handle the custom type
			if originType != originTypes.Custom {
				continue
			} else if len(accessOriginStrings)<=i+1 {
				paramErrors = append(paramErrors, "“Custom” requires a parameter")
			} else {
				arguments.CustomAccessOrigin = accessOriginStrings[i+1]
			}

			//Once custom has been handled, there is no point on checking anything else
			break
		}
	}

	//If there are errors, output them and the defaults and exit
	if len(paramErrors)>0 {
		println(strings.Join(paramErrors, "\n"))
		println("\nDefaults:")
		flag.PrintDefaults()
		println("") //Add extra space at the end
		return
	}

	//Create the handler and start listening for requests
	InitServer(arguments) //CustomCallback
	http.HandleFunc("/", arguments.forwardHandlerWrapper)
	http.ListenAndServe(":"+strconv.Itoa(arguments.LocalPort), nil)
}

//Helper function for determining if the first string in an array starts with another string (case insensative)
//If "trueOnEmpty", returns true if searchStrings is null or empty
func FirstStringStartsWith(searchStrings []string, startString string, trueOnEmpty bool) bool {
	if searchStrings == nil || len(searchStrings) == 0 {
		return trueOnEmpty
	}

	return (len(searchStrings[0]) >= len(startString) && strings.ToLower(searchStrings[0][:len(startString)]) == strings.ToLower(startString))
}
