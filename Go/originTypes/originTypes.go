//Enum of Origin Types and their mappings
package originTypes
type OriginType int
const (
	Original OriginType = iota
	OriginalSwapped
	LocalHost
	RemoteHost
	RequestOrigin
	AcceptAll
	Custom
)
var Mapping map[string]OriginType = map[string]OriginType{"original":Original, "originalswapped":OriginalSwapped, "localhost":LocalHost, "remotehost":RemoteHost, "requestorigin":RequestOrigin, "acceptall":AcceptAll, "custom":Custom}