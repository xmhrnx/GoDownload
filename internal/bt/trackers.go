package bt

// publicTrackers are added to public torrents whose own trackers are dead or
// few, which is the usual reason a well-seeded torrent finds no peers.
var publicTrackers = [][]string{
	{"udp://tracker.opentrackr.org:1337/announce"},
	{"udp://open.stealth.si:80/announce"},
	{"udp://tracker.torrent.eu.org:451/announce"},
	{"udp://exodus.desync.com:6969/announce"},
	{"udp://open.demonii.com:1337/announce"},
	{"udp://tracker.openbittorrent.com:6969/announce"},
	{"https://tracker.tamersunion.org:443/announce"},
}
