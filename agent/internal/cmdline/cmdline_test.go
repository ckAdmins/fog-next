package cmdline

import "testing"

func TestParseString(t *testing.T) {
	tests := []struct {
		line string
		want Params
	}{
		{
			line: "BOOT_IMAGE=/vmlinuz fog_server=http://10.0.0.1 fog_action=deploy fog_host=aa:bb:cc:dd:ee:ff",
			want: Params{FogServer: "http://10.0.0.1", FogAction: "deploy", FogHost: "aa:bb:cc:dd:ee:ff", FogTUI: true, FogAuto: true},
		},
		{
			line: "fog_server=http://fog fog_debug=1",
			want: Params{FogServer: "http://fog", FogDebug: true, FogTUI: true, FogAuto: true},
		},
		{
			line: "",
			want: Params{FogTUI: true, FogAuto: true},
		},
		{
			line: "fog_server=http://fog fog_tui=0",
			want: Params{FogServer: "http://fog", FogTUI: false, FogAuto: true},
		},
		{
			line: "fog_server=http://fog fog_tui=false",
			want: Params{FogServer: "http://fog", FogTUI: false, FogAuto: true},
		},
		{
			line: "fog_server=http://fog fog_auto=0",
			want: Params{FogServer: "http://fog", FogAuto: false, FogTUI: true},
		},
		{
			line: "fog_server=http://fog fog_auto=false",
			want: Params{FogServer: "http://fog", FogAuto: false, FogTUI: true},
		},
		{
			line: "fog_server=http://fog fog_auto=1",
			want: Params{FogServer: "http://fog", FogAuto: true, FogTUI: true},
		},
	}
	for _, tc := range tests {
		got := ParseString(tc.line)
		if *got != tc.want {
			t.Errorf("ParseString(%q) = %+v, want %+v", tc.line, *got, tc.want)
		}
	}
}
