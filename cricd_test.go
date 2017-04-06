package cricd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func init() {
	os.Setenv("DEBUG", "true")
}

func TestNewConfigSet(t *testing.T) {
	assert := assert.New(t)
	_ = os.Setenv("EVENTAPI_IP", "1.1.1.1")
	_ = os.Setenv("EVENTAPI_PORT", "1234")
	_ = os.Setenv("ENTITYSTORE_IP", "1.2.3.4")
	_ = os.Setenv("ENTITYSTORE_PORT", "4567")

	c := NewConfig()

	assert.Equal(c.eventAPIIP, "1.1.1.1", "Failed to get EVENTAPI_IP from ENV VAR")
	assert.Equal(c.eventAPIPort, "1234", "Failed to get EVENTAPI_PORT from ENV VAR")
	assert.Equal(c.entityStoreIP, "1.2.3.4", "Failed to get ENTITYSTORE_IP from ENV VAR")
	assert.Equal(c.entityStorePort, "4567", "Failed to get ENTITYSTORE_PORT from ENV VAR")
}

func TestNewConfigNotSet(t *testing.T) {
	os.Clearenv()
	assert := assert.New(t)
	c := NewConfig()
	assert.Equal(c.eventAPIIP, "localhost", "Failed to get EVENTAPI_IP from ENV VAR")
	assert.Equal(c.eventAPIPort, "4567", "Failed to get EVENTAPI_PORT from ENV VAR")
	assert.Equal(c.entityStoreIP, "localhost", "Failed to get ENTITYSTORE_IP from ENV VAR")
	assert.Equal(c.entityStorePort, "1337", "Failed to get ENTITYSTORE_PORT from ENV VAR")
}

type PlayerEndpointTest struct {
	name          string
	input         Player
	serverRes     []Player
	serverResRaw  string
	serverResCode int
	output        Player
	ok            bool
}

func TestPlayer_Get(t *testing.T) {
	os.Clearenv()
	c := NewConfig()
	var tst = []PlayerEndpointTest{
		{
			name:      "Bad Player",
			input:     NewPlayer(c),
			serverRes: []Player{},
			output:    NewPlayer(c),
			ok:        false},
		{
			name:      "Good Player",
			input:     Player{ID: 0, Name: "Ryan Scott", DateOfBirth: time.Date(1970, 1, 1, 1, 1, 1, 1, time.Local), Gender: "male", conf: c},
			serverRes: []Player{{ID: 27, Name: "Ryan Scott", DateOfBirth: time.Date(1970, 1, 1, 1, 1, 1, 1, time.Local), Gender: "male", conf: c}},
			output:    Player{ID: 27, Name: "Ryan Scott", DateOfBirth: time.Date(1970, 1, 1, 1, 1, 1, 1, time.Local), Gender: "male", conf: c},
			ok:        true},
		{
			name:         "Broken Player",
			input:        Player{ID: 0, Name: "Ryan Foo", DateOfBirth: time.Date(1970, 1, 1, 1, 1, 1, 1, time.Local), Gender: "male", conf: c},
			serverResRaw: "{fjfdsao}",
			serverRes:    []Player{},
			output:       Player{ID: 0, Name: "Ryan Foo", DateOfBirth: time.Date(1970, 1, 1, 1, 1, 1, 1, time.Local), Gender: "male", conf: c},
			ok:           false},
	}
	var res []byte
	serv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, string(res))
	}))
	defer serv.Close()

	for _, ts := range tst {
		// Set the response
		if ts.serverResRaw == "" {
			res, _ = json.Marshal(ts.serverRes)
		} else {
			res = []byte(ts.serverResRaw)
		}
		c.playersURL = serv.URL

		ok, _ := ts.input.Get()

		assert.Equal(t, ts.output, ts.input, "Expected player not received")
		assert.Equal(t, ts.ok, ok, "Expected a false OK")
	}
}

func TestPlayer_Create(t *testing.T) {
	os.Clearenv()
	c := NewConfig()
	var tst = []PlayerEndpointTest{
		{
			name:      "Bad Player",
			input:     NewPlayer(c),
			serverRes: []Player{},
			output:    NewPlayer(c),
			ok:        false},
		{
			name:      "Existing Player",
			input:     Player{ID: 27, Name: "Ryan Scott", DateOfBirth: time.Date(1970, 1, 1, 1, 1, 1, 1, time.Local), Gender: "male", conf: c},
			serverRes: []Player{},
			output:    Player{ID: 27, Name: "Ryan Scott", DateOfBirth: time.Date(1970, 1, 1, 1, 1, 1, 1, time.Local), Gender: "male", conf: c},
			ok:        false},
	}
	var res []byte
	serv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, string(res))
	}))
	defer serv.Close()

	for _, ts := range tst {
		fmt.Printf("Running test: TestPlayer_Create-%s \n", ts.name)
		// Set the response
		if ts.serverResRaw == "" {
			res, _ = json.Marshal(ts.serverRes)
		} else {
			res = []byte(ts.serverResRaw)
		}
		c.playersURL = serv.URL

		ok, _ := ts.input.Create()

		assert.Equal(t, ts.output, ts.input, "Expected player not received for")
		assert.Equal(t, ts.ok, ok, "Expected a false OK")
	}

}

func TestPlayer_GetOrCreatePlayer(t *testing.T) {
	type fields struct {
		ID          int
		Name        string
		DateOfBirth time.Time
		Gender      string
		conf        *Config
	}
	tests := []struct {
		name    string
		fields  fields
		wantOk  bool
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Player{
				ID:          tt.fields.ID,
				Name:        tt.fields.Name,
				DateOfBirth: tt.fields.DateOfBirth,
				Gender:      tt.fields.Gender,
				conf:        tt.fields.conf,
			}
			gotOk, err := p.GetOrCreatePlayer()
			if (err != nil) != tt.wantErr {
				t.Errorf("Player.GetOrCreatePlayer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotOk != tt.wantOk {
				t.Errorf("Player.GetOrCreatePlayer() = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}
