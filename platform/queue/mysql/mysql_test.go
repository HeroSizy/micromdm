package mysql

import (
	"context"
	"fmt"
//	"io/ioutil"
//	"os"
	"testing"

	"github.com/satori/go.uuid"
	
	"github.com/go-kit/kit/log"
	"github.com/kolide/kit/dbutil"
	_ "github.com/go-sql-driver/mysql"
	"github.com/micromdm/micromdm/platform/queue"
	"github.com/micromdm/micromdm/mdm"
)

func TestNext_Error(t *testing.T) {
	store := setupDB(t)
	dc := &queue.DeviceCommand{DeviceUDID: "TestDevice"}
	dc.Commands = append(dc.Commands, queue.Command{UUID: "xCmd"})
	dc.Commands = append(dc.Commands, queue.Command{UUID: "yCmd"})
	dc.Commands = append(dc.Commands, queue.Command{UUID: "zCmd"})
	ctx := context.Background()
	if err := store.Save(ctx, dc); err != nil {
		t.Fatal(err)
	}

	resp := mdm.Response {
		UDID:        dc.DeviceUDID,
		CommandUUID: "xCmd",
		Status:      "Error",
	}
	for range dc.Commands {
		cmd, err := store.nextCommand(ctx, resp)
		if err != nil {
			t.Fatalf("expected nil, but got err: %s", err)
		}
		if cmd == nil {
			t.Fatal("expected cmd but got nil")
		}

		if have, errd := cmd.UUID, resp.CommandUUID; have == errd {
			t.Error("got back command which previously failed")
		}
	}
}

func TestNext_NotNow(t *testing.T) {
	store := setupDB(t)

	dc := &queue.DeviceCommand{DeviceUDID: "TestDevice"}
	dc.Commands = append(dc.Commands, queue.Command{UUID: "xCmd"})
	dc.Commands = append(dc.Commands, queue.Command{UUID: "yCmd"})
	ctx := context.Background()
	if err := store.Save(ctx, dc); err != nil {
		t.Fatal(err)
	}
	tf := func(t *testing.T) {

		resp := mdm.Response{
			UDID:        dc.DeviceUDID,
			CommandUUID: "yCmd",
			Status:      "NotNow",
		}
		cmd, err := store.nextCommand(ctx, resp)

		if err != nil {
			t.Fatalf("expected nil, but got err: %s", err)
		}

		resp = mdm.Response{
			UDID:        dc.DeviceUDID,
			CommandUUID: cmd.UUID,
			Status:      "NotNow",
		}

		cmd, err = store.nextCommand(ctx, resp)

		if err != nil {
			t.Fatalf("expected nil, but got err: %s", err)
		}
		if cmd != nil {
			t.Error("Got back a notnowed command.")
		}
	}

	t.Run("withManyCommands", tf)
	dc.Commands = []queue.Command{{UUID: "xCmd"}}
	if err := store.Save(ctx, dc); err != nil {
		t.Fatal(err)
	}
	t.Run("withOneCommand", tf)
}

func TestNext_Idle(t *testing.T) {
	store := setupDB(t)

	dc := &queue.DeviceCommand{DeviceUDID: "TestDevice"}
	dc.Commands = append(dc.Commands, queue.Command{UUID: "xCmd"})
	dc.Commands = append(dc.Commands, queue.Command{UUID: "yCmd"})
	dc.Commands = append(dc.Commands, queue.Command{UUID: "zCmd"})
	ctx := context.Background()
	if err := store.Save(ctx, dc); err != nil {
		t.Fatal(err)
	}

	resp := mdm.Response{
		UDID:        dc.DeviceUDID,
		CommandUUID: "xCmd",
		Status:      "Idle",
	}
	for i := range dc.Commands {
		cmd, err := store.nextCommand(ctx, resp)
		if err != nil {
			t.Fatalf("expected nil, but got err: %s", err)
		}
		if cmd == nil {
			t.Fatal("expected cmd but got nil")
		}

		if have, want := cmd.UUID, dc.Commands[i].UUID; have != want {
			t.Errorf("have %s, want %s, index %d", have, want, i)
		}
	}
}

func TestNext_zeroCommands(t *testing.T) {
	store := setupDB(t)

	dc := &queue.DeviceCommand{DeviceUDID: "TestDevice"}
	ctx := context.Background()
	if err := store.Save(ctx, dc); err != nil {
		t.Fatal(err)
	}

	var allStatuses = []string{
		"Acknowledged",
		"NotNow",
	}

	for _, s := range allStatuses {
		t.Run(s, func(t *testing.T) {
			resp := mdm.Response{CommandUUID: s, Status: s}
			cmd, err := store.nextCommand(ctx, resp)
			if err != nil {
				t.Errorf("expected nil, but got err: %s", err)
			}
			if cmd != nil {
				t.Errorf("expected nil cmd but got %s", cmd.UUID)
			}
		})
	}

}


func TestSave_Insert(t *testing.T) {
	store := setupDB(t)
	ctx := context.Background()
	
	u1 := uuid.NewV4()
	uniqueString := fmt.Sprintf("%s", u1)

	commandExists, err := store.ContainsCommand(ctx, uniqueString)
	if err != nil {
		t.Fatal(err)
	}
	if commandExists != false {
		t.Errorf("Expects new command with new uuid to not exist yet")
	}

	command := queue.Command{UUID: uniqueString}
	
	
	dc := &queue.DeviceCommand{DeviceUDID: "TestDevice"}
	dc.Commands = append(dc.Commands, command)
	if err := store.Save(ctx, dc); err != nil {
		t.Fatal(err)
	}
	
	commandExists, err = store.ContainsCommand(ctx, uniqueString)
	if commandExists != true {
		t.Errorf("Expects old command with new uuid to not exist yet")
	}
}



func Test_SaveCommand(t *testing.T) {
	store := setupDB(t)

	dc := &queue.DeviceCommand{DeviceUDID: "TestDevice"}
	ctx := context.Background()
	if err := store.Save(ctx, dc); err != nil {
		t.Fatal(err)
	}

}


func setupDB(t *testing.T) *Store {
	// https://stackoverflow.com/a/23550874/464016
	db, err := dbutil.OpenDBX(
		"mysql",
		"micromdm:micromdm@tcp(127.0.0.1:3306)/micromdm_test?parseTime=true",
		//"host=127.0.0.1 port=3306 user=micromdm dbname=micromdm_test password=micromdm sslmode=disable",
		dbutil.WithLogger(log.NewNopLogger()),
		dbutil.WithMaxAttempts(1),
	)
	
	if err != nil {
		t.Fatal(err)
	}

	store := &Store{db: db, logger: log.NewNopLogger()}
	_,err = db.Exec(`TRUNCATE TABLE device_commands;`)
	//store.NewQueue(db, nil)
	return store
}
