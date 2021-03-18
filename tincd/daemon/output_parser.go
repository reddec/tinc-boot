package daemon

//go:generate events-gen -k -E Events -p daemon -o events.go .
import (
	"regexp"
)

// Example log line:
//
//		Sending DEL_SUBNET to everyone (BROADCAST): 11 3f17d1ce hubreddecnet_PEN005 6e:6a:5e:26:39:d2#10

var (
	addSubnetPattern = regexp.MustCompile(`ADD_SUBNET\s+from\s+([^\s]+)\s+\(([^\s]+)\s+port\s+(\d+)\)\:\s+\d+\s+[\w\d]+\s+([^\s]+)\s+([^#]+)`)
	delSubnetPattern = regexp.MustCompile(`DEL_SUBNET\s+[^:]+:\s+\d+\s+[\w\d]+\s+([^\s]+)\s+([^#]+)`)
	readyPattern     = regexp.MustCompile(`^Ready$`)
)

// event:"SubnetAdded"
type EventSubnetAdded struct {
	Advertising struct {
		Node string
		Host string
		Port string
	}
	Peer struct {
		Node   string
		Subnet string
	}
}

func (event *EventSubnetAdded) Parse(line string) bool {
	match := addSubnetPattern.FindAllStringSubmatch(line, -1)
	if len(match) == 0 {
		return false
	}
	groups := match[0]
	if len(groups) != 6 {
		return false
	}
	event.Advertising.Node = groups[1]
	event.Advertising.Host = groups[2]
	event.Advertising.Port = groups[3]
	event.Peer.Node = groups[4]
	event.Peer.Subnet = groups[5]
	return true
}

func IsSubnetAdded(line string) *EventSubnetAdded {
	var esr EventSubnetAdded
	if esr.Parse(line) {
		return &esr
	}
	return nil
}

// event:"SubnetRemoved"
type EventSubnetRemoved struct {
	Peer struct {
		Node   string
		Subnet string
	}
}

func (event *EventSubnetRemoved) Parse(line string) bool {
	match := delSubnetPattern.FindAllStringSubmatch(line, -1)
	if len(match) == 0 {
		return false
	}
	groups := match[0]
	if len(groups) != 3 {
		return false
	}
	event.Peer.Node = groups[1]
	event.Peer.Subnet = groups[2]
	return true
}

func IsSubnetRemoved(line string) *EventSubnetRemoved {
	var esr EventSubnetRemoved
	if esr.Parse(line) {
		return &esr
	}
	return nil
}

// event:"Ready"
type EventReady struct {
}

func (event *EventReady) Parse(line string) bool {
	return readyPattern.MatchString(line)
}

func IsReady(line string) *EventReady {
	var esr EventReady
	if esr.Parse(line) {
		return &esr
	}
	return nil
}
