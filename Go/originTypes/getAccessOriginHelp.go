package originTypes

import "fmt"

func GetAccessOriginHelp(PrintAlso bool) string {

	const str string = `
******************************Access Origin help*******************************
Information about how the reply “Access-Control-Allow-Origin” is determined

This parameter takes a list of origin types (below) that define the precedence
of variables in which to determine the “Access-Control-Allow-Origin”. Types
are evaluated left to right, and the first evaluated Type which has a value
is what is used. Types can be used multiple times.

Origin Types (case insensitive):
* Original
    * This is the “Access-Control-Allow-Origin” returned from the remote host
* OriginalSwapped
    * This takes the “Original” origin type and performs a host swap on it
* LocalHost
    * This uses the value of the -LocalHost parameter. It always has a result
    * If the -LocalPort parameter is not the default, a colon and it will be
      appended to the -RemoteHost. It assumes http as the protocol.
* RemoteHost
    * This is evaluated in the same manner as the -LocalHost, but also uses the
      -RemoteProtocol parameter
* RequestOrigin
    * The value of the “Origin” request header, if given
* AcceptAll
    * All domains are valid (“*” is passed)
* Custom
    * Follow this with your custom value as the next item in the list. It
      cannot contain commas. There can only be 1 Custom type in the list

Example: OriginalSwapped,RequestOrigin,Custom=http://www.foobar.com

Default (When Blank) = OriginalSwapped,RequestOrigin,LocalHost

If none of the given origin types yield a result, the header is not included
*******************************************************************************
`

	if PrintAlso {
		fmt.Println(str)
	}

	return str
}
