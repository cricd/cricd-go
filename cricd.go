package cricd

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	cache "github.com/patrickmn/go-cache"
)

const DateFormat = "2006-01-02"

var playerCache = cache.New(5*time.Minute, 30*time.Second)
var teamCache = cache.New(5*time.Minute, 30*time.Second)
var matchCache = cache.New(5*time.Minute, 30*time.Second)

type Team struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	conf *Config
}

type Player struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	DateOfBirth time.Time `json:"dateOfBirth"`
	Gender      string    `json:"gender"`
	conf        *Config
}

type Match struct {
	ID              int       `json:"id"`
	HomeTeam        Team      `json:"homeTeam"`
	AwayTeam        Team      `json:"awayTeam"`
	StartDate       time.Time `json:"startDate"`
	NumberOfInnings int       `json:"numberOfInnings"`
	LimitedOvers    int       `json:"limitedOvers"`
	conf            *Config
}

type Innings struct {
	Number       int
	BattingTeam  Team
	FieldingTeam Team
}

// Config defines the configuration required to use the cricd package
type Config struct {
	eventAPIIP      string
	eventAPIPort    string
	entityStoreIP   string
	entityStorePort string
	playersURL      string
	teamsURL        string
	matchesURL      string
}

// NewPlayer returns a new player, using the config provided or uses logical defaults
func NewPlayer(c *Config) Player {
	var p Player
	if c != nil {
		p.conf = c
	} else {
		p.conf = NewConfig()

	}
	return p
}

// TODO: Test me
func (p *Player) GetOrCreatePlayer() (ok bool, err error) {
	k, e := p.Get()

	if e != nil {
		log.WithFields(log.Fields{"error": err}).Error("Failed to get player")
		return false, err
	}
	if !k {
		k, e := p.Create()
		if e != nil {
			log.WithFields(log.Fields{"error": err}).Error("Failed to create player")
			return false, err
		}
		if !k {
			log.Error("Failed to create player without error")
			return false, nil
		}
	}
	log.Debugf("Returning player with Name: %s ID#: %d", p.Name, p.ID)
	return true, nil
}

// TODO: Test me
func (p *Player) Create() (ok bool, err error) {
	params := url.Values{
		"name": {p.Name},
	}

	log.Debugf("Sending request to create players to: %s", p.conf.playersURL)
	log.Debugf("Using the following params to create players: %s", params)
	res, err := http.PostForm(p.conf.playersURL, params)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Failed to call create player endpoint")
		return false, err
	}
	if res.StatusCode != http.StatusCreated {
		log.WithFields(log.Fields{"response": res.Status, "code": res.StatusCode}).Error("Got not OK response from creating player")
		return false, fmt.Errorf("Got non OK status code when creating player: %d", res.StatusCode)
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Failed to read body from create player endpoint")
		return false, err
	}
	var cp Player
	err = json.Unmarshal(body, &cp)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Failed to unmarshal JSON from create player endpoint")
		return false, err
	}
	// If we have a non-null Player
	if (Player{}) != cp {
		p.ID = cp.ID
		return true, nil
	}

	log.Error("Create player endpoint returned no players")
	return false, fmt.Errorf("Failed to return any players when creating a player")

}

// TODO: Test me
func (p *Player) Get() (ok bool, err error) {

	// Try hit the cache first
	player, found := playerCache.Get(p.Name)
	if found {
		log.Debugf("Returning player from the player cache: %d - %s", p.ID, p.Name)
		p.ID = player.(Player).ID
		return true, nil
	}
	req, err := http.NewRequest("GET", p.conf.playersURL, nil)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Failed to create request to get from players endpoint")
		return false, err
	}
	// Build query string
	q := req.URL.Query()
	q.Add("name", p.Name)
	req.URL.RawQuery = q.Encode()
	// Send request
	log.Debugf("Sending request to get player to: %s", req.URL)

	client := &http.Client{Timeout: 1 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Failed to send request to get players endpoint")
		return false, err
	}
	if res.StatusCode != (http.StatusOK) {
		log.WithFields(log.Fields{"response": res.Status, "code": res.StatusCode}).Error("Got not OK response from getting players")
		return false, fmt.Errorf("Received a non OK status code when getting players, got: %d", res.StatusCode)
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Failed to read body from get player endpoint")
		return false, err
	}
	var t []Player
	err = json.Unmarshal(body, &t)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Failed to unmarshal JSON from get player endpoint")
		return false, err
	}

	// More than one player we'll take the first
	if len(t) > 1 {
		log.WithFields(log.Fields{"players": len(t)}).Info("More than one team returned from get player endpoint, using the first")
		p.ID = t[0].ID
		log.Infof("Got player with ID#: %d", p.ID)
		playerCache.Set(t[0].Name, t[0], cache.DefaultExpiration)
		return true, nil
		// If there's exactly one, then we're good
	} else if len(t) == 1 {
		p.ID = t[0].ID
		log.Debugf("Returning team from get player endpoint, ID#: %d Name: %s", t[0].ID, t[0].Name)
		playerCache.Set(t[0].Name, t[0], cache.DefaultExpiration)
		return true, nil
	} else {
		// Otherwise we didn't get anything
		log.Info("Not returning any players from get player endpoint")
		return false, nil
	}

}

// NewTeam returns a new team, using the config provided or uses logical defaults
func NewTeam(c *Config) Team {
	var t Team
	if c != nil {
		t.conf = c
	} else {
		t.conf = NewConfig()

	}
	return t
}

// TODO: Test me
func (t *Team) Create() (ok bool, err error) {
	params := url.Values{
		"name": {t.Name},
	}

	log.Debugf("Sending request to create team to: %s", t.conf.teamsURL)
	log.Debugf("Using the following params to create team: %s", params)
	res, err := http.PostForm(t.conf.teamsURL, params)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Failed to call create team endpoint")
		return false, err
	}
	if res.StatusCode != http.StatusCreated {
		log.WithFields(log.Fields{"response": res.Status, "code": res.StatusCode}).Error("Got not OK response from creating team")
		return false, fmt.Errorf("Got non OK status code when creating team: %d", res.StatusCode)
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Failed to read body from create team endpoint")
		return false, err
	}
	var ct Team
	err = json.Unmarshal(body, &ct)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Failed to unmarshal JSON from create team endpoint")
		return false, err
	}
	// If we got a legitimate team
	if ct.ID != 0 {
		t.ID = ct.ID
		log.Debugf("Returning team from create team endpoint, ID#: %d Name: %s", ct.ID, ct.Name)
		return true, nil
	}

	log.Errorf("Failed to create team from create team endpoint")
	return false, fmt.Errorf("Failed to create team from create team endpoint")

}

//TODO: Test me
func (t *Team) Get() (ok bool, err error) {
	// Try hit the cache first
	tm, found := teamCache.Get(t.Name)
	if found {
		log.Debugf("Returning team from the team cache: %s", t.Name)
		t.ID = tm.(Team).ID
		return true, nil
	}
	req, err := http.NewRequest("GET", t.conf.teamsURL, nil)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Failed to get team from team endpoint")
		return false, err
	}
	// Build query string
	q := req.URL.Query()
	q.Add("name", t.Name)
	req.URL.RawQuery = q.Encode()
	// Send request
	log.Debugf("Sending request to get team to: %s", req.URL)

	client := &http.Client{Timeout: 1 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Failed to send request to get team endpoint")
		return false, err
	}
	if res.StatusCode != (http.StatusOK) {
		log.WithFields(log.Fields{"response": res.Status, "code": res.StatusCode}).Error("Got not OK response from getting teams")
		return false, fmt.Errorf("Received a non OK status code when getting teams, got: %d", res.StatusCode)
	}
	body, err := ioutil.ReadAll(res.Body)
	var ct []Team
	err = json.Unmarshal(body, &ct)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Failed to unmarshal JSON from get team endpoint")
		return false, err
	}
	// If we got more than one team
	if len(ct) > 1 {
		log.WithFields(log.Fields{"teams": len(ct)}).Info("More than one team returned from  get team endpoint")
		t.ID = ct[0].ID
		teamCache.Set(ct[0].Name, ct[0], cache.DefaultExpiration)
		return true, nil
	} else if len(ct) == 1 {
		t.ID = ct[0].ID
		teamCache.Set(ct[0].Name, ct[0], cache.DefaultExpiration)
		log.Debugf("Returning team from get team endpoint, ID#: %d Name: %s", t.ID, t.Name)
		return true, nil
	} else {
		log.Info("Not returning any teams from get team endpoint")
		return false, nil
	}
}

// NewMatch returns a new match, using the config provided or uses logical defaults
func NewMatch(c *Config) Match {
	var m Match
	if c != nil {
		m.conf = c
	} else {
		m.conf = NewConfig()

	}
	return m
}

// TODO: Test me
func (m *Match) Create() (ok bool, err error) {
	log.Debugf("Creating match between %s and %s", m.HomeTeam, m.AwayTeam)
	params := url.Values{
		"homeTeam":        {strconv.Itoa(m.HomeTeam.ID)},
		"awayTeam":        {strconv.Itoa(m.AwayTeam.ID)},
		"numberOfInnings": {strconv.Itoa(m.NumberOfInnings)},
		"limitedOvers":    {strconv.Itoa(m.LimitedOvers)},
		"startDate":       {m.StartDate.Format(DateFormat)},
	}

	log.Debugf("Sending request to create match to: %s", m.conf.matchesURL)
	log.Debugf("Using the following params to create match: %s", params)
	res, err := http.PostForm(m.conf.matchesURL, params)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Failed to call create match endpoint")
		return false, err
	}
	if res.StatusCode != http.StatusCreated {
		log.WithFields(log.Fields{"response": res.Status, "code": res.StatusCode}).Error("Got not OK response from creating match")
		return false, fmt.Errorf("Got non OK status code when creating match: %d", res.StatusCode)
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Failed to read body from create match endpoint")
		return false, err
	}

	var matches Match
	err = json.Unmarshal(body, &matches)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Failed to unmarshal JSON from create match endpoint")
		return false, err
	}
	// If we got a legitimate match
	if matches.ID != 0 {
		m.ID = matches.ID
		log.Debugf("Returning match from create match endpoint, ID#:", m.ID)
		return true, nil
	}
	log.Errorf("Not returning any matches from create match endpoint ")
	return false, nil

}

// TODO: Test me
func (m *Match) Get() (ok bool, err error) {

	// Try hit the cache first
	matchKey := base64.StdEncoding.EncodeToString([]byte(m.AwayTeam.Name + m.HomeTeam.Name + m.StartDate.Format(DateFormat)))
	match, found := matchCache.Get(matchKey)
	if found {
		log.Debugf("Returning match from the match cache: %s", matchKey)
		m.ID = match.(Match).ID
		return true, nil
	}

	req, err := http.NewRequest("GET", m.conf.matchesURL, nil)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Failed to create request to get from match endpoint")
	}
	// Build query string
	q := req.URL.Query()
	q.Add("homeTeam", strconv.Itoa(m.HomeTeam.ID))
	q.Add("awayTeam", strconv.Itoa(m.AwayTeam.ID))
	q.Add("numberOfInnings", strconv.Itoa(m.NumberOfInnings))
	q.Add("limitedOvers", strconv.Itoa(m.LimitedOvers))
	q.Add("startDate", m.StartDate.Format(DateFormat))
	req.URL.RawQuery = q.Encode()
	// Send request
	log.Debugf("Sending request to get match to: %s", req.URL)

	client := &http.Client{Timeout: 1 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Failed to send request to get match endpoint")
		return false, err
	}
	if res.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{"response": res.Status, "code": res.StatusCode}).Error("Got not OK response from getting match")
		return false, fmt.Errorf("Received a non OK status code when getting match, got: %d", res.StatusCode)
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Failed to read body from get match endpoint")
		return false, err
	}

	var matches []Match
	err = json.Unmarshal(body, &matches)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Failed to unmarshal JSON from get match endpoint")
		return false, err
	}

	if len(matches) > 1 {
		log.WithFields(log.Fields{"teams": len(matches)}).Info("More than one team returned from get match endpoint")
		m.ID = matches[0].ID
		matchCache.Set(matchKey, matches[0], cache.DefaultExpiration)
		return true, nil
	} else if len(matches) == 1 {
		m.ID = matches[0].ID
		matchCache.Set(matchKey, matches[0], cache.DefaultExpiration)
		log.Debugf("Returning match from get match endpoint, ID#: %d between %s and %s on: %s", matches[0].ID, matches[0].HomeTeam, matches[0].AwayTeam, matches[0].StartDate)
		return true, nil
	} else {
		log.Debugf("Not returning any matches from get match endpoint ")
		return false, nil
	}

}

// NewConfig returns a configuration instance and tries to get the values from ENV vars, otherwise sets it to logical defaults
func NewConfig() *Config {
	var c Config
	eaIP := os.Getenv("EVENTAPI_IP")
	if eaIP != "" {
		c.eventAPIIP = eaIP
		log.WithFields(log.Fields{"value": eaIP}).Debug("Found ENV var for event API IP")

	} else {
		log.WithFields(log.Fields{"value": "EVENTAPI_IP"}).Debug("Unable to find env var, using default `localhost`")
		c.eventAPIIP = "localhost"
	}

	eaPort := os.Getenv("EVENTAPI_PORT")
	if eaPort != "" {
		c.eventAPIPort = eaPort
		log.WithFields(log.Fields{"value": eaPort}).Debug("Found ENV var for event API port")

	} else {
		log.WithFields(log.Fields{"value": "EVENTAPI_PORT"}).Debug("Unable to find env var, using default `4567`")
		c.eventAPIPort = "4567"

	}

	etURL := os.Getenv("ENTITYSTORE_IP")
	if etURL != "" {
		c.entityStoreIP = etURL
		log.WithFields(log.Fields{"value": etURL}).Debug("Found ENV var for entity store IP")

	} else {
		log.WithFields(log.Fields{"value": "ENTITYSTORE_IP"}).Debug("Unable to find env var, using default `localhost`")
		c.entityStoreIP = "localhost"
	}

	etPort := os.Getenv("ENTITYSTORE_PORT")
	if etPort != "" {
		c.entityStorePort = etPort
		log.WithFields(log.Fields{"value": etPort}).Debug("Found ENV var for entity store port")

	} else {
		log.WithFields(log.Fields{"value": "ENTITYSTORE_PORT"}).Debug("Unable to find env var, using default `1337`")
		c.entityStorePort = "1337"
	}

	// Set up the endpoints
	c.matchesURL = fmt.Sprintf("http://%s:%s/matches", c.entityStoreIP, c.entityStorePort)
	log.Debugf("Setting matches URL to: %s", c.matchesURL)

	c.playersURL = fmt.Sprintf("http://%s:%s/players", c.entityStoreIP, c.entityStorePort)
	log.Debugf("Setting players URL to: %s", c.playersURL)

	c.teamsURL = fmt.Sprintf("http://%s:%s/teams", c.entityStoreIP, c.entityStorePort)
	log.Debugf("Setting teams URL to: %s", c.teamsURL)

	return &c
}

// NewDelivery returns a new match, using the config provided or uses logical defaults
func NewDelivery(c *Config) Delivery {
	var d Delivery
	if c != nil {
		d.conf = c
	} else {
		d.conf = NewConfig()

	}
	return d
}

type Delivery struct {
	MatchID   int    `json:"match"`
	EventType string `json:"eventType"`
	Timestamp string `json:"timestamp"`
	Ball      struct {
		BattingTeam  Team `json:"battingTeam"`
		FieldingTeam Team `json:"fieldingTeam"`
		Innings      int  `json:"innings"`
		Over         int  `json:"over"`
		Ball         int  `json:"ball"`
	} `json:"ball"`
	Runs    int `json:"runs"`
	Batsmen struct {
		Striker    Player `json:"striker"`
		NonStriker Player `json:"nonStriker"`
	} `json:"batsmen"`
	Bowler  Player  `json:"bowler"`
	Fielder *Player `json:"fielder,omitempty"`
	conf    *Config
}

// Push pushes a delivery to the Event API for persistence
func (d *Delivery) Push() (ok bool, err error) {
	etURL := fmt.Sprintf("http://%s:%s/event", d.conf.eventAPIIP, d.conf.eventAPIPort)
	log.Debugf("Sending request to Event API at %s", etURL)
	json, err := json.Marshal(d)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Failed to marshal delivery to json")
		return false, err
	}
	req, err := http.NewRequest("POST", etURL, bytes.NewBuffer(json))
	req.Header.Set("Content-Type", "application/json")
	params := url.Values{}
	params.Set("NextBall", "false")
	client := &http.Client{Timeout: 2 * time.Second}

	res, err := client.Do(req)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Failed to send to event api")
		return false, err
	}
	if res.StatusCode != http.StatusCreated {
		log.WithFields(log.Fields{"response": res.Status, "code": res.StatusCode}).Error("Got not OK response from event API")
		return false, fmt.Errorf("Received a %d code - %s from event store api", res.StatusCode, res.Status)
	}
	defer res.Body.Close()
	return true, nil
}

func init() {
	debug := os.Getenv("DEBUG")
	if debug == "true" {
		log.WithFields(log.Fields{"value": "DEBUG"}).Info("Setting log level to debug")
		log.SetLevel(log.DebugLevel)
	} else {
		log.Info("Setting log level to info")
		log.SetLevel(log.InfoLevel)
	}
	log.SetOutput(os.Stdout)
}
