package warcrumb

import (
	"fmt"
	"os"
	"path"
	"testing"
)

func TestRead(t *testing.T) {

	tests := []struct {
		name     string
		filePath string
		//wantRep Replay
		isReforged bool
		wantErr    bool
	}{
		{"1.01", "1.01-LeoLaporte_vs_Ghostridah_crazy.w3g", false, false},
		{"1.18", "1.18-replayspl_4105_MKpowa_KrawieC..w3g", false, false},
		{"1.18 vs computers", "W3R-118-Archie(HU) & Ezzo(HU) vs Computer (Insane)(RND) & Computer (Insane)(RND).w3g", false, false},
		{"1.30 grub", "W3R-22259-Grubby(O) vs Happy(UD).w3g", false, false},
		{"1.31 just before reforged", "W3R-28524-Lyn(O) vs LawLiet(NE).w3g", false, false},
		{"my first win", "FirstWin.w3g", true, false},
		{"my second win", "secondwin.w3g", true, false},
		{"reforged Pudge wars", "reforgedPudgeWars.w3g", true, false},
		{"reforged private game", "refTest.w3g", true, false},
		{"reforged Tower game", "refTower.w3g", true, false},
		{"reforged Tower rush", "refTowerRush.w3g", true, false},
		{"reforged lotr", "lotr.w3g", true, false},
		{"reforged offline", "refOffline.w3g", true, false},
		{"reforged LAN 2 player", "2pLan.w3g", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fmt.Println(tt.filePath)
			f, err := os.Open(path.Join("testReplays", tt.filePath))
			if err != nil {
				t.Errorf("Could not open test replay: %v", err)
			}
			rep, err := ParseReplayDebug(f)
			fmt.Println(rep.GameOptions.GameName, rep.GameOptions.MapName, rep.Version)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseReplay() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil && tt.isReforged != rep.isReforged {
				t.Errorf("ParseReplay() gotIsReforged = %t, want %t", tt.isReforged, rep.isReforged)
			}

			//if !reflect.DeepEqual(gotRep, tt.wantRep) {
			//	t.Errorf("ParseReplay() gotRep = %v, want %v", gotRep, tt.wantRep)
			//}
		})
	}
}
