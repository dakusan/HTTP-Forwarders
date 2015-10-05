[HTTP-Forwarders v1.0.0<br>
http://www.castledragmire.com/Projects/HTTP-Forwarders](http://www.castledragmire.com/Projects/HTTP-Forwarders)

# Go and PHP HTTP forwarder/proxies

So I needed an http proxy for a project I am currently working on which lets me completely mirror a website on my own domain via forwarding (in this instance, for breaking cross-domain scripting). The only useful thing I was quickly able to find on the internet was [Rob Thomson’s PHP proxy](at https://code.google.com/p/php-proxy/). However, it doesn’t support file upload passthrough, so I went ahead and added that. Unfortunately, the latency added by passing everything through PHP was much too high.

So I decided to make something in GoLang (Google’s Go), as it is fast and great at handling concurrency. I’ve been hacking at it all day today, and it’s ended up taking a lot more time than I originally planned, but I think it’s pretty perfect now.

**Note: The Go version is much more advanced, reliable, and fast.**

## Go version

It supports passthrough forwarding of **POST, GET, files, cookies, https**, and anything else that comes up on an http connection (though https takes some tinkering sometimes). It also supports gzip and deflate encoding (it needs to get everything into plain text for modification). It works as a man-in-the-middle, receiving a request, and then transmitting that request to the designation. It runs on a port you specify, and it replaces all instances of the source domain with the destination domain when sending to the destination, and vice-versa when receiving it back.

#### Host name swapping
This is a little weird due to web server constraints and default port settings. The process for it is as follows

1. Determine which direction you are going (you->remote or remote->you), which we will call source->destination
2. Generate the destination string that will be replacing the source strings
  * This is the destination domain optionally followed by a colon and the port. The colon+port is only added if the port is not the default port for the protocol (80 for http, 443 for https)
  * The “you” side doesn’t actually know what protocol it’s using, as Go does not seem to be supplying that information (I’m sure it’s somewhere), so if either 80 or 443 is used, it is considered to be using the default.
3. Replace *SourceHost:SourcePort* with the replacement string in #2
4. Replace *SourceHost* with the replacement string in #2

#### Customizing
There is also a **custom.go** file which lets you add your own rules for the server. It has 3 callback functions
* **InitServer**
  * Called before any other “http” calls
  * Contains an example of *http.HandleFunc* in which it calls a custom handler for the url */custom/&#42;*
* **ModifyForwarderRequest**
  * Called after request object and headers are created, but before the http request
  * Contains an example of adding a cookie to the request
* **ModifyForwarderReply**
  * Called after the response headers+cookies have been created and the response content host-swapped, but before the header and contents are written back
  * Contains 2 examples
    1. Adding a cookie to send to the client
    2. Adding content to the HTML body

#### The command line parameters are as follows:
* **LocalPort** (Optional)
  * The local port to which you connect to forward a request
  * Default: 8080
* **RemoteHost** (Required)
  * The domain host that requests are forwarded to
  * Default: &lt;blank>
* **RemotePort** (Optional)
  * The port for the remote domain
  * Default: 80
* **RemoteProtocol** (Optional)
  * The protocol (http/https) for the remote domain
  * Default: http
* **LocalHost** (Optional)
  * The host domain you are connecting to locally
  * In the response body, the RemoteHost is replaced by the LocalHost
  * The default, used if blank, is the domain pulled from your first http request
  * Default: &lt;blank>

#### Go installation
1. Install go, if not already installed
2. In a command shell:
  1. ./make
  2. ./forward (*./forward.exe on cygwin/windows)*
  3. Read the list of parameters and run “forward” again with your needed ones

## PHP installation
1. Copy the PHP folder to your web server (PHP is required, obviously)
2. Rename htaccess.txt to .htaccess (for apache servers)
3. In the index.php, edit the settings under “config settings”
4. Navigate to the index.php in a web browser, and voila

## Licenses
**PHP version**: This is under the GNU General Public License version 2<br>
**Go version**: This is under my license, which is the 4 clause BSD
