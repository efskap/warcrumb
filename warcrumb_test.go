package main

import (
	"fmt"
	"os"
	"path"
	"testing"
)

func TestRead(t *testing.T) {

	tests := []struct {
		name    string
        filePath string
		//wantRep Replay
		isReforged bool
		wantErr bool
	}{
       {"1.01", "1.01-LeoLaporte_vs_Ghostridah_crazy.w3g",  false,false},
		{"1.18", "1.18-replayspl_4105_MKpowa_KrawieC..w3g", false, false},
		{"my first win", "FirstWin.w3g", true, false},
		{"my second win", "secondwin.w3g", true, false},
		{"reforged Pudge wars", "reforgedPudgeWars.w3g", true, false},
		{"reforged private game", "refTest.w3g", true, false},
		{"reforged Tower game", "refTower.w3g", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fmt.Println(tt.filePath)
			f, err := os.Open(path.Join("testReplays",tt.filePath))
			if err != nil {
				t.Errorf("Could not open test replay: %v", err)
			}
			rep, err := Read(f)
			if (err != nil) != tt.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			fmt.Printf("%+v\n", rep)

			if err == nil &&  tt.isReforged != rep.isReforged {
				t.Errorf("Read() gotIsReforged = %t, want %t", tt.isReforged, rep.isReforged)
			}

			//if !reflect.DeepEqual(gotRep, tt.wantRep) {
			//	t.Errorf("Read() gotRep = %v, want %v", gotRep, tt.wantRep)
			//}
		})
	}
}


