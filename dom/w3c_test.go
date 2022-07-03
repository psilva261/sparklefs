package dom

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
)

func TestAll(t *testing.T) {
	files, err := ioutil.ReadDir("test/w3c/level1/core")
	if err != nil {
		t.Fatalf(err.Error())
	}

	for _, f := range files {
		switch f.Name() {
		case "hc_documentgetelementsbytagnamelength.js", "hc_documentgetelementsbytagnametotallength.js", "hc_nodeappendchildnodeancestor.js", "hc_nodeelementnodetype.js", "hc_nodeinsertbeforenodeancestor.js", "hc_nodeinsertbeforerefchildnonexistent.js", "hc_noderemovechildoldchildnonexistent.js", "hc_nodereplacechildnodeancestor.js", "hc_nodereplacechildoldchildnonexistent.js", "hc_textindexsizeerrnegativeoffset.js", "hc_textindexsizeerroffsetoutofbounds.js":
			continue
		case "hc_characterdatareplacedataexceedslengthofarg.js":
			continue
		}
		if strings.Contains(f.Name(), "exception") {
			continue
		}
		if strings.Contains(f.Name(), "err") {
			continue
		}
		if strings.HasSuffix(f.Name(), ".js") {
			fmt.Println(f.Name())
			err := W3C(f.Name())
			if err != nil {
				t.Fatalf(err.Error())
			}
		}
	}
}
