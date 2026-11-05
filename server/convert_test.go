package main

import "testing"

const H = `\\fileserver.example.com`

// lnk is the expected output: the path as inline code, linked to openfolder://.
func lnk(text, urlPath string) string {
	return "[`" + text + "`](openfolder://fileserver.example.com" + urlPath + ")"
}

func TestConvert(t *testing.T) {
	c := NewConverter([]string{"fileserver.example.com"})

	cases := []struct{ name, in, want string }{
		{"simple",
			H + `\projects\2026`,
			lnk(H+`\projects\2026`, "/projects/2026")},
		{"trailing backslash kept as typed",
			H + `\share\`,
			lnk(H+`\share\`, "/share")},
		{"spaces to end of line",
			`It is here: ` + H + `\Projects 2026\Final report`,
			`It is here: ` + lnk(H+`\Projects 2026\Final report`, "/Projects%202026/Final%20report")},
		{"trailing period trimmed",
			`See ` + H + `\a b\c.`,
			`See ` + lnk(H+`\a b\c`, "/a%20b/c") + `.`},
		{"brackets kept when balanced",
			H + `\Project (old)\x`,
			lnk(H+`\Project (old)\x`, "/Project%20%28old%29/x")},
		{"surrounding parenthesis trimmed",
			`(see ` + H + `\a b)`,
			`(see ` + lnk(H+`\a b`, "/a%20b") + `)`},
		{"double quotes with words after",
			`Folder "` + H + `\Team A\Data" please check`,
			`Folder "` + lnk(H+`\Team A\Data`, "/Team%20A/Data") + `" please check`},
		{"german quotes",
			`„` + H + `\Team A“ ok`,
			`„` + lnk(H+`\Team A`, "/Team%20A") + `“ ok`},
		{"backticks",
			"Path: `" + H + `\Team A\x` + "` thanks",
			`Path: ` + lnk(H+`\Team A\x`, "/Team%20A/x") + ` thanks`},
		{"two paths on one line",
			H + `\a b ` + H + `\c`,
			lnk(H+`\a b`, "/a%20b") + ` ` + lnk(H+`\c`, "/c")},
		{"file url",
			`file://fileserver.example.com/Team%20A/x`,
			lnk(H+`\Team A\x`, "/Team%20A/x")},
		{"other server untouched",
			`\\other.example.org\share\x`,
			`\\other.example.org\share\x`},
		{"look-alike host untouched",
			`\\fileserver.example.com.evil.test\share\x`,
			`\\fileserver.example.com.evil.test\share\x`},
		{"host only untouched",
			H,
			H},
		{"case insensitive",
			`\\FILESERVER.example.com\x`,
			"[`\\\\FILESERVER.example.com\\x`](openfolder://FILESERVER.example.com/x)"},
		{"markdown chars shown literally",
			H + `\my_folder\[1]`,
			lnk(H+`\my_folder\[1]`, "/my_folder/%5B1%5D")},
		{"fenced code untouched",
			"```\n" + H + "\\a\n```\nthen " + H + `\b`,
			"```\n" + H + "\\a\n```\nthen " + lnk(H+`\b`, "/b")},
		{"multi line",
			"Hello\n" + H + "\\a b\nBye",
			"Hello\n" + lnk(H+`\a b`, "/a%20b") + "\nBye"},
		{"legacy escaped link upgraded",
			`[\\\\fileserver.example.com\\share\\](openfolder://fileserver.example.com/share)`,
			lnk(H+`\share\`, "/share")},
		{"legacy escaped link with underscore upgraded",
			`[\\\\fileserver.example.com\\my\_folder](openfolder://fileserver.example.com/my_folder)`,
			lnk(H+`\my_folder`, "/my_folder")},
		{"other markdown link untouched",
			`[Docs](https://example.com/a_b)`,
			`[Docs](https://example.com/a_b)`},
	}
	for _, tc := range cases {
		got := c.Convert(tc.in)
		if got != tc.want {
			t.Errorf("%s:\n in:   %s\n got:  %s\n want: %s", tc.name, tc.in, got, tc.want)
		}
		if again := c.Convert(got); again != got {
			t.Errorf("%s: not idempotent:\n first:  %s\n second: %s", tc.name, got, again)
		}
	}
}

func TestMultipleServers(t *testing.T) {
	c := NewConverter(splitServers("fileserver.example.com, nas.example.org"))
	in := "\\\\nas.example.org\\backup\n\\\\fileserver.example.com\\x"
	want := "[`\\\\nas.example.org\\backup`](openfolder://nas.example.org/backup)\n" +
		"[`\\\\fileserver.example.com\\x`](openfolder://fileserver.example.com/x)"
	if got := c.Convert(in); got != want {
		t.Errorf("got:  %s\nwant: %s", got, want)
	}
}

func TestNoServersConfigured(t *testing.T) {
	if c := NewConverter(splitServers("  ,  ")); c != nil {
		t.Fatal("expected nil converter for an empty server list")
	}
	var c *Converter
	if got := c.Convert(H + `\x`); got != H+`\x` {
		t.Errorf("nil converter changed the message: %s", got)
	}
}
