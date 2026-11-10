package doctor

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"unicode/utf16"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

func Decode(data []byte, choice string) (string, string, bool, error) {
	if len(data) > MaxBytes {
		return "", "", false, fmt.Errorf("CSV Doctor supports files up to 16 MiB. No file was changed")
	}
	if bytes.HasPrefix(data, []byte{0xff, 0xfe, 0, 0}) || bytes.HasPrefix(data, []byte{0, 0, 0xfe, 0xff}) {
		return "", "", false, fmt.Errorf("UTF-32 is not supported; convert it explicitly in another tool")
	}
	enc, certain := choice, choice != "" && choice != "auto"
	if !certain {
		switch {
		case bytes.HasPrefix(data, []byte{0xef, 0xbb, 0xbf}):
			enc, certain = "utf8bom", true
		case bytes.HasPrefix(data, []byte{0xff, 0xfe}):
			enc, certain = "utf16le", true
		case bytes.HasPrefix(data, []byte{0xfe, 0xff}):
			enc, certain = "utf16be", true
		case looksUTF16(data, true):
			enc = "utf16le"
		case looksUTF16(data, false):
			enc = "utf16be"
		case utf8.Valid(data):
			enc, certain = "utf8", true
		default:
			enc = "windows1252"
		}
	}
	var text string
	switch enc {
	case "utf8", "utf8bom":
		data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})
		if !utf8.Valid(data) {
			return "", enc, certain, fmt.Errorf("Invalid UTF-8 bytes. Choose the source encoding explicitly; invalid bytes are never replaced silently")
		}
		text = string(data)
	case "utf16le", "utf16be":
		bom := []byte{0xff, 0xfe}
		if enc == "utf16be" {
			bom = []byte{0xfe, 0xff}
		}
		data = bytes.TrimPrefix(data, bom)
		if len(data)%2 != 0 {
			return "", enc, certain, fmt.Errorf("UTF-16 has an incomplete code unit")
		}
		units := make([]uint16, len(data)/2)
		for i := range units {
			if enc == "utf16le" {
				units[i] = binary.LittleEndian.Uint16(data[i*2:])
			} else {
				units[i] = binary.BigEndian.Uint16(data[i*2:])
			}
		}
		for i := 0; i < len(units); i++ {
			u := units[i]
			if u >= 0xd800 && u <= 0xdbff {
				if i+1 >= len(units) || units[i+1] < 0xdc00 || units[i+1] > 0xdfff {
					return "", enc, certain, fmt.Errorf("UTF-16 contains an unpaired surrogate")
				}
				i++
			} else if u >= 0xdc00 && u <= 0xdfff {
				return "", enc, certain, fmt.Errorf("UTF-16 contains an unpaired surrogate")
			}
		}
		text = string(utf16.Decode(units))
	case "windows1252":
		for _, b := range data {
			if b == 0x81 || b == 0x8d || b == 0x8f || b == 0x90 || b == 0x9d {
				return "", enc, certain, fmt.Errorf("Undefined Windows-1252 bytes detected. This encoding cannot safely represent the input")
			}
		}
		out, err := charmap.Windows1252.NewDecoder().Bytes(data)
		if err != nil {
			return "", enc, certain, err
		}
		text = string(out)
	default:
		return "", enc, false, fmt.Errorf("Unsupported source encoding")
	}
	if bytes.IndexByte([]byte(text), 0) >= 0 {
		return "", enc, certain, fmt.Errorf("NUL characters detected. Choose the correct encoding or a delimited text file")
	}
	return text, enc, certain, nil
}
func looksUTF16(data []byte, little bool) bool {
	if len(data) < 4 || len(data)%2 != 0 {
		return false
	}
	n := len(data)
	if n > 4096 {
		n = 4096
	}
	zeros, other := 0, 0
	side := 1
	if !little {
		side = 0
	}
	for i := 0; i < n; i += 2 {
		if data[i+side] == 0 {
			zeros++
		}
		if data[i+1-side] == 0 {
			other++
		}
	}
	return zeros*4 > n && other*10 < n
}

func lineStyle(s string) string {
	crlf, lf, cr := 0, 0, 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\r' {
			if i+1 < len(s) && s[i+1] == '\n' {
				crlf++
				i++
			} else {
				cr++
			}
		} else if s[i] == '\n' {
			lf++
		}
	}
	count := 0
	style := "None"
	for _, v := range []struct {
		n int
		s string
	}{{crlf, "CRLF"}, {lf, "LF"}, {cr, "CR"}} {
		if v.n > 0 {
			count++
			style = v.s
		}
	}
	if count > 1 {
		return "Mixed"
	}
	return style
}
