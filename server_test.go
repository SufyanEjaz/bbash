//
// Copyright (c) 2021-present Sonatype, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/sonatype-nexus-community/bbash/internal/db"
	"github.com/sonatype-nexus-community/bbash/internal/types"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
	"net"
	"net/http"
	"net/http/httptest"
	url2 "net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

var now = time.Now()

func resetEnvVariable(t *testing.T, variableName, originalValue string) {
	if originalValue == "" {
		assert.NoError(t, os.Unsetenv(variableName))
	} else {
		assert.NoError(t, os.Setenv(variableName, originalValue))
	}
}

func resetEnvVar(t *testing.T, envVarName, origValue string) {
	if origValue != "" {
		assert.NoError(t, os.Setenv(envVarName, origValue))
	} else {
		assert.NoError(t, os.Unsetenv(envVarName))
	}
}

func resetEnvVarPGHost(t *testing.T, origEnvPGHost string) {
	resetEnvVar(t, envPGHost, origEnvPGHost)
}

// multiple mock kludge counters
var insertBugGuidCount int
var priorScoreCallCount float64
var updateScoreLastDelta float64

type MockBBashDB struct {
	t                *testing.T
	assertParameters bool

	mockDb *sql.DB

	migrateDbSourceURL string
	migrateDbErr       error

	getSCPPs    []types.SourceControlProviderStruct
	getSCPPsErr error

	insertCampaignParam *types.CampaignStruct
	insertCampaignGuid  string
	insertCampaignErr   error

	updateCampaignParam *types.CampaignStruct
	updateCampaignGuid  string
	updateCampaignErr   error

	getCampaignParam  string
	getCampaignResult *types.CampaignStruct
	getCampaignErr    error

	getActiveCampaignsParam     time.Time
	getActiveCampaignsParamSkip bool
	getActiveCampaignsResult    []types.CampaignStruct
	getActiveCampaignsErr       error

	getCampaignsResult []types.CampaignStruct
	getCampaignsErr    error

	insertOrganizationParam *types.OrganizationStruct
	insertOrganizationGuid  string
	insertOrganizationErr   error

	getOrganizationsResult []types.OrganizationStruct
	getOrganizationsErr    error

	deleteOrgSCPName      string
	deleteOrgOrgName      string
	deleteOrgRowsAffected int64
	deleteOrgErr          error

	validOrgParam  *types.ScoringMessage
	validOrgResult bool
	validOrgErr    error

	partiesToScoreMsg     *types.ScoringMessage
	partiesToScoreNowSkip bool
	partiesToScoreNow     time.Time
	partiesToScoreResult  []types.ParticipantStruct
	partiesToScoreErr     error

	selectPointValueMsg      *types.ScoringMessage
	selectPointValueCampaign string
	selectPointValueBugType  string
	selectPointValueResult   float64

	updateScoreParticipant *types.ParticipantStruct
	updateScoreDelta       int
	updateScoreErr         error

	priorScoreParticipant *types.ParticipantStruct
	priorScoreMsg         *types.ScoringMessage
	priorScoreResult      float64

	insertScoreEvtPartier   *types.ParticipantStruct
	insertScoreEvtMsg       *types.ScoringMessage
	insertScoreEvtNewPoints int
	insertScoreEvtErr       error

	insertParticipantPartier  *types.ParticipantStruct
	insertParticipantGuid     string
	insertParticipantJoinedAt time.Time
	insertParticipantErr      error

	updateParticipantPartier      *types.ParticipantStruct
	updateParticipantRowsAffected int64
	updateParticipantErr          error

	selectPartDetailCampName  string
	selectPartDetailSCPName   string
	selectPartDetailLoginName string
	selectPartDetailResult    *types.ParticipantStruct
	selectPartDetailErr       error

	selectPartInCampCamp   string
	selectPartInCampResult []types.ParticipantStruct
	selectPartInCampErr    error

	deletePartCampaign  string
	deletePartSCPName   string
	deletePartLoginName string
	deletePartGuid      string
	deletePartErr       error

	insertTeamTm   *types.TeamStruct
	insertTeamGuid string
	insertTeamErr  error

	updatePartTeamTeamName     string
	updatePartTeamCampaignName string
	updatePartTeamSCPName      string
	updatePartTeamLoginName    string
	updatePartTeamRowsAffected int64
	updatePartTeamErr          error

	insertBugBug  *types.BugStruct
	insertBugGuid string
	insertBugErr  error

	updateBugBug          *types.BugStruct
	updateBugRowsAffected int64
	updateBugErr          error

	selectBugsResult []types.BugStruct
	selectBugsErr    error

	selectPoll    types.Poll
	selectPollErr error
	updatePoll    types.Poll
	updatePollErr error
}

func (m MockBBashDB) GetDb() (db *sql.DB) {
	return m.mockDb
}

func (m MockBBashDB) MigrateDB(migrateSourceURL string) error {
	if m.assertParameters {
		assert.Equal(m.t, m.migrateDbSourceURL, migrateSourceURL)
	}
	return m.migrateDbErr
}

func (m MockBBashDB) GetSourceControlProviders() (scps []types.SourceControlProviderStruct, err error) {
	return m.getSCPPs, m.getSCPPsErr
}

func (m MockBBashDB) InsertCampaign(campaign *types.CampaignStruct) (guid string, err error) {
	if m.assertParameters {
		assert.Equal(m.t, m.insertCampaignParam, campaign)
	}
	return m.insertCampaignGuid, m.insertCampaignErr
}

func (m MockBBashDB) UpdateCampaign(campaign *types.CampaignStruct) (guid string, err error) {
	if m.assertParameters {
		assert.Equal(m.t, m.updateCampaignParam, campaign)
	}
	return m.updateCampaignGuid, m.updateCampaignErr
}

func (m MockBBashDB) GetCampaign(campaignName string) (campaign *types.CampaignStruct, err error) {
	if m.assertParameters {
		assert.Equal(m.t, m.getCampaignParam, campaignName)
	}
	return m.getCampaignResult, m.getCampaignErr
}

func (m MockBBashDB) GetCampaigns() (campaigns []types.CampaignStruct, err error) {
	return m.getCampaignsResult, m.getCampaignsErr
}

func (m MockBBashDB) GetActiveCampaigns(now time.Time) (activeCampaigns []types.CampaignStruct, err error) {
	if m.assertParameters {
		if !m.getActiveCampaignsParamSkip {
			assert.Equal(m.t, m.getActiveCampaignsParam, now)
		}
	}
	return m.getActiveCampaignsResult, m.getActiveCampaignsErr
}

func (m MockBBashDB) InsertOrganization(organization *types.OrganizationStruct) (guid string, err error) {
	if m.assertParameters {
		assert.Equal(m.t, m.insertOrganizationParam, organization)
	}
	return m.insertOrganizationGuid, m.insertOrganizationErr
}

func (m MockBBashDB) GetOrganizations() (organizations []types.OrganizationStruct, err error) {
	return m.getOrganizationsResult, m.getOrganizationsErr
}

func (m MockBBashDB) DeleteOrganization(scpName, orgName string) (rowsAffected int64, err error) {
	if m.assertParameters {
		assert.Equal(m.t, m.deleteOrgSCPName, scpName)
		assert.Equal(m.t, m.deleteOrgOrgName, orgName)
	}
	return m.deleteOrgRowsAffected, m.deleteOrgErr
}

func (m MockBBashDB) ValidOrganization(msg *types.ScoringMessage) (orgExists bool, err error) {
	if m.assertParameters {
		assert.Equal(m.t, m.validOrgParam, msg)
	}
	return m.validOrgResult, m.validOrgErr
}

func (m MockBBashDB) SelectParticipantsToScore(msg *types.ScoringMessage, now time.Time) (participantsToScore []types.ParticipantStruct, err error) {
	if m.assertParameters {
		assert.Equal(m.t, m.partiesToScoreMsg, msg)
		// some callers use dynamic Time.now() value, so we can't validate exact value
		if !m.partiesToScoreNowSkip {
			assert.Equal(m.t, m.partiesToScoreNow, now)
		}
	}
	return m.partiesToScoreResult, m.partiesToScoreErr
}

func (m MockBBashDB) SelectPointValue(msg *types.ScoringMessage, campaignName, bugType string) (pointValue float64) {
	if m.assertParameters {
		assert.Equal(m.t, m.selectPointValueMsg, msg)
		assert.Equal(m.t, m.selectPointValueCampaign, campaignName)
		assert.Equal(m.t, m.selectPointValueBugType, bugType)
	}
	return m.selectPointValueResult
}

func (m MockBBashDB) UpdateParticipantScore(participant *types.ParticipantStruct, delta float64) (err error) {
	if m.assertParameters {
		// multiple mock kludge
		if priorScoreCallCount == 0 {
			assert.Equal(m.t, m.updateScoreParticipant, participant)
			assert.Equal(m.t, m.updateScoreDelta, delta)
		}
	}
	updateScoreLastDelta = delta
	return m.updateScoreErr
}

func (m MockBBashDB) SelectPriorScore(participantToScore *types.ParticipantStruct, msg *types.ScoringMessage) (oldPoints float64) {
	if m.assertParameters {
		// multiple mock kludge
		if priorScoreCallCount == 0 {
			assert.Equal(m.t, m.priorScoreParticipant, participantToScore)
			assert.Equal(m.t, m.priorScoreMsg, msg)
		}
	}
	// kludge to support multiple calls to mock. maybe
	scoreToReturn := m.priorScoreResult + priorScoreCallCount
	priorScoreCallCount++
	return scoreToReturn
}

func (m MockBBashDB) InsertScoringEvent(participantToScore *types.ParticipantStruct, msg *types.ScoringMessage, newPoints float64) (err error) {
	if m.assertParameters {
		// multiple mock kludge
		if priorScoreCallCount == 0 {
			assert.Equal(m.t, m.insertScoreEvtPartier, participantToScore)
			assert.Equal(m.t, m.insertScoreEvtMsg, msg)
			assert.Equal(m.t, m.insertScoreEvtNewPoints, newPoints)
		}
	}
	return m.insertScoreEvtErr
}

func (m MockBBashDB) InsertParticipant(participant *types.ParticipantStruct) (err error) {
	if m.assertParameters {
		assert.Equal(m.t, m.insertParticipantPartier, participant)
	}
	// alter the passed in struct with newly created mock values
	participant.ID = m.insertParticipantGuid
	participant.Score = 0
	participant.JoinedAt = m.insertParticipantJoinedAt
	return m.insertParticipantErr
}

func (m MockBBashDB) SelectParticipantDetail(campaignName, scpName, loginName string) (participant *types.ParticipantStruct, err error) {
	if m.assertParameters {
		assert.Equal(m.t, m.selectPartDetailCampName, campaignName)
		assert.Equal(m.t, m.selectPartDetailSCPName, scpName)
		assert.Equal(m.t, m.selectPartDetailLoginName, loginName)
	}
	return m.selectPartDetailResult, m.selectPartDetailErr
}

func (m MockBBashDB) DeleteParticipant(campaign, scpName, loginName string) (participantId string, err error) {
	if m.assertParameters {
		assert.Equal(m.t, m.deletePartCampaign, campaign)
		assert.Equal(m.t, m.deletePartSCPName, scpName)
		assert.Equal(m.t, m.deletePartLoginName, loginName)
	}
	return m.deletePartGuid, m.deletePartErr
}

func (m MockBBashDB) SelectParticipantsInCampaign(campaignName string) (participants []types.ParticipantStruct, err error) {
	if m.assertParameters {
		assert.Equal(m.t, m.selectPartInCampCamp, campaignName)
	}
	return m.selectPartInCampResult, m.selectPartInCampErr
}

func (m MockBBashDB) InsertTeam(team *types.TeamStruct) (err error) {
	if m.assertParameters {
		assert.Equal(m.t, m.insertTeamTm, team)
	}
	// alter the passed in struct with newly created mock values
	team.Id = m.insertTeamGuid
	return m.insertTeamErr
}

func (m MockBBashDB) UpdateParticipant(participant *types.ParticipantStruct) (rowsAffected int64, err error) {
	if m.assertParameters {
		assert.Equal(m.t, m.updateParticipantPartier, participant)
	}
	return m.updateParticipantRowsAffected, m.updateParticipantErr
}

func (m MockBBashDB) UpdateParticipantTeam(teamName, campaignName, scpName, loginName string) (rowsAffected int64, err error) {
	if m.assertParameters {
		assert.Equal(m.t, m.updatePartTeamTeamName, teamName)
		assert.Equal(m.t, m.updatePartTeamCampaignName, campaignName)
		assert.Equal(m.t, m.updatePartTeamSCPName, scpName)
		assert.Equal(m.t, m.updatePartTeamLoginName, loginName)
	}
	return m.updatePartTeamRowsAffected, m.updatePartTeamErr
}

func (m MockBBashDB) InsertBug(bug *types.BugStruct) (err error) {
	if m.assertParameters {
		// only validate the first calls parameter. maybe later, could change mocks to support lists to validate
		if insertBugGuidCount == 0 {
			assert.Equal(m.t, m.insertBugBug, bug)
		}
	}
	// alter the passed in struct with newly created mock values
	if insertBugGuidCount == 0 {
		bug.Id = m.insertBugGuid
	} else {
		bug.Id = fmt.Sprintf("%s%d", m.insertBugGuid, insertBugGuidCount)
	}
	insertBugGuidCount++
	return m.insertBugErr
}

func (m MockBBashDB) UpdateBug(bug *types.BugStruct) (rowsAffected int64, err error) {
	if m.assertParameters {
		assert.Equal(m.t, m.updateBugBug, bug)
	}
	return m.updateBugRowsAffected, m.updateBugErr
}

func (m MockBBashDB) SelectBugs() (bugs []types.BugStruct, err error) {
	return m.selectBugsResult, m.selectBugsErr
}

func (m MockBBashDB) NewPoll() types.Poll {
	return db.NewPoll()
}

func (m MockBBashDB) UpdatePoll(poll *types.Poll) (err error) {
	if m.assertParameters {
		assert.Equal(m.t, m.updatePoll, poll)
	}
	return m.updatePollErr
}

func (m MockBBashDB) SelectPoll(poll *types.Poll) (err error) {
	if m.assertParameters {
		assert.Equal(m.t, m.selectPoll, poll)
	}
	return m.selectPollErr
}

var _ db.IBBashDB = (*MockBBashDB)(nil)

func newMockDb(t *testing.T) (mockDbIF *MockBBashDB) {
	mockDbIF = &MockBBashDB{
		t:                t,
		assertParameters: true,
	}
	// reset loop kludge counters
	insertBugGuidCount = 0
	priorScoreCallCount = 0
	updateScoreLastDelta = 0

	logger = zaptest.NewLogger(t)

	// side effect: set up the postgresDB var
	postgresDB = mockDbIF
	return
}

func TestZapLoggerFilterSkipsELB(t *testing.T) {
	req := httptest.NewRequest("", "/", nil)
	req.Header.Set("User-Agent", "bing ELB-HealthChecker yadda")
	logger := zaptest.NewLogger(t)
	result := ZapLoggerFilterAwsElb(logger)

	//handlerFunc := func(next echo.HandlerFunc) echo.HandlerFunc {
	//	return func(c echo.Context) error {
	//		return nil
	//	}
	//}
	//r2 := result(handlerFunc)
	//assert.Nil(t, result)
	// @TODO figure out how to test these hoops
	result(nil)
}

func TestMainDBPingError(t *testing.T) {
	errRecovered = nil
	origEnvPGHost := os.Getenv(envPGHost)
	defer func() {
		resetEnvVarPGHost(t, origEnvPGHost)
	}()
	assert.NoError(t, os.Setenv(envPGHost, "bogus-db-hostname"))

	defer func() {
		errRecovered = nil
	}()

	main()

	assert.True(t, strings.HasPrefix(errRecovered.Error(), "failed to ping database. host: bogus-db-hostname, port: "))
}

func TestMainDBMigrateError(t *testing.T) {
	errRecovered = nil
	origEnvPGHost := os.Getenv(envPGHost)
	defer func() {
		resetEnvVarPGHost(t, origEnvPGHost)
	}()
	assert.NoError(t, os.Setenv(envPGHost, "localhost"))

	// setup mock db endpoint
	l, err := net.Listen("tcp", "localhost:0")
	assert.NoError(t, err)
	defer func(l net.Listener) {
		_ = l.Close()
	}(l)
	//goland:noinspection HttpUrlsUsage
	u, err := url2.Parse("http://" + l.Addr().String())
	assert.NoError(t, err)
	freeLocalPort := u.Port()
	assert.NoError(t, os.Setenv(envPGPort, freeLocalPort))
	go func() {
		conn, err := l.Accept()
		assert.NoError(t, err)
		defer func(conn net.Conn) {
			_ = conn.Close()
		}(conn)
		b := make([]byte, 0, 512)
		count, err := conn.Read(b)
		_, _ = conn.Write(b)
		assert.NoError(t, err)
		assert.Equal(t, count, 0)
	}()

	defer func() {
		errRecovered = nil
	}()

	main()

	assert.True(t, strings.HasPrefix(errRecovered.Error(), "failed to ping database. host: localhost, port: "+freeLocalPort), errRecovered.Error())
}

func TestMigrateDB(t *testing.T) {
	dbMock := newMockDb(t)
	dbMock.migrateDbSourceURL = "testMigrateUrl"
	assert.NoError(t, dbMock.MigrateDB("testMigrateUrl"))
}

func TestSetupRoutes(t *testing.T) {
	e := echo.New()

	logger = zaptest.NewLogger(t)

	customRouteCount := setupRoutes(e, "myBuildInfoMsg")
	routes := e.Routes()
	// when using "groups", extra "default" routes are automatically added by echo
	//assert.Equal(t, 22, len(routes))
	// Out main() method will only print "custom" routes, ignoring defaults added by echo. such defaults are still
	// included in the "total" route count below
	assert.Equal(t, 200, len(routes))

	assert.Equal(t, 23, customRouteCount)
}

const timeLayout = "2006-01-02T15:04:05.000Z"

var testStartOn time.Time
var testEndOn time.Time

func init() {
	var err error
	testStartOn, err = time.Parse(timeLayout, "2021-11-01T12:00:00.000Z")
	if err != nil {
		panic(err)
	}
	testEndOn, err = time.Parse(timeLayout, "2021-11-02T12:00:00.000Z")
	if err != nil {
		panic(err)
	}
}
func setupMockContextCampaign(campaignName string) (c echo.Context, rec *httptest.ResponseRecorder, expectedCampaign *types.CampaignStruct) {
	c, rec = setupMockContextCampaignWithBody(campaignName, fmt.Sprintf("{ \"startOn\": \"%s\", \"endOn\": \"%s\"}",
		testStartOn.Format(timeLayout), testEndOn.Format(timeLayout)))
	expectedCampaign = &types.CampaignStruct{
		Name:    campaignName,
		StartOn: testStartOn,
		EndOn:   testEndOn,
	}
	return
}
func setupMockContextCampaignWithBody(campaignName, bodyCampaign string) (c echo.Context, rec *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest("", "/", strings.NewReader(bodyCampaign))
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.SetParamNames(ParamCampaignName)
	c.SetParamValues(campaignName)
	return
}

func TestAddCampaignEmptyName(t *testing.T) {
	campaignName := " "
	c, rec, testCampaign := setupMockContextCampaign(campaignName)

	mock := newMockDb(t)
	mock.insertCampaignParam = testCampaign

	expectedError := fmt.Errorf("invalid parameter %s: %s", ParamCampaignName, "")

	assert.NoError(t, addCampaign(c))
	assert.Equal(t, http.StatusBadRequest, c.Response().Status)
	assert.Equal(t, expectedError.Error(), rec.Body.String())
}

func TestGetCampaignsError(t *testing.T) {
	c, rec := setupMockContext()

	mock := newMockDb(t)
	forcedError := fmt.Errorf("forced campaign error")
	mock.getCampaignsErr = forcedError

	assert.EqualError(t, getCampaigns(c), forcedError.Error())
	assert.Equal(t, 0, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

func TestGetCampaigns(t *testing.T) {
	c, rec := setupMockContext()

	mock := newMockDb(t)
	mock.getCampaignsResult = []types.CampaignStruct{{
		ID:           campaignId,
		Name:         campaign,
		CreatedOn:    time.Time{},
		CreatedOrder: 1,
		StartOn:      now,
		EndOn:        now,
	}}
	assert.NoError(t, getCampaigns(c))
	assert.Equal(t, http.StatusOK, c.Response().Status)
	expectedCampaigns := []types.CampaignStruct{
		{ID: campaignId, Name: campaign, CreatedOn: time.Time{}, CreatedOrder: 1, StartOn: now, EndOn: now},
	}
	jsonExpectedCampaign, err := json.Marshal(expectedCampaigns)
	assert.NoError(t, err)
	assert.Equal(t, string(jsonExpectedCampaign)+"\n", rec.Body.String())
}

func TestGetActiveCampaignsError(t *testing.T) {
	c, rec, testCampaign := setupMockContextCampaign(campaign)

	mock := newMockDb(t)
	mock.getActiveCampaignsResult = []types.CampaignStruct{*testCampaign}

	forcedError := fmt.Errorf("forced campaign error")
	mock.getActiveCampaignsErr = forcedError
	// caller users Time.now(), so don't assert time parameter
	mock.getActiveCampaignsParamSkip = true
	assert.NoError(t, getActiveCampaigns(c))
	assert.Equal(t, http.StatusBadRequest, c.Response().Status)
	assert.Equal(t, forcedError.Error(), rec.Body.String())
}

func TestGetActiveCampaigns(t *testing.T) {
	c, rec, _ := setupMockContextCampaign(campaign)

	mock := newMockDb(t)
	mock.getActiveCampaignsResult = []types.CampaignStruct{
		{ID: campaignId, Name: campaign, StartOn: now, EndOn: now},
	}
	// caller users Time.now(), so don't assert time parameter
	mock.getActiveCampaignsParamSkip = true

	assert.NoError(t, getActiveCampaigns(c))
	assert.Equal(t, http.StatusOK, c.Response().Status)

	jsonExpectedCampaign, err := json.Marshal(mock.getActiveCampaignsResult)
	assert.NoError(t, err)
	assert.Equal(t, string(jsonExpectedCampaign)+"\n", rec.Body.String())
}

func TestAddCampaignErrorReadingCampaignFromRequestBody(t *testing.T) {
	c, rec := setupMockContextCampaignWithBody(campaign, "")

	assert.EqualError(t, addCampaign(c), "EOF")
	assert.Equal(t, 0, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

func TestAddCampaignError(t *testing.T) {
	c, rec, testCampaign := setupMockContextCampaign(campaign)

	mock := newMockDb(t)
	mock.insertCampaignParam = testCampaign
	forcedError := fmt.Errorf("forced campaign error")
	mock.insertCampaignErr = forcedError

	assert.EqualError(t, addCampaign(c), forcedError.Error())
	assert.Equal(t, 0, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

func TestAddCampaign(t *testing.T) {
	c, rec, testCampaign := setupMockContextCampaign(campaign)

	mock := newMockDb(t)
	mock.insertCampaignParam = testCampaign
	mock.insertCampaignGuid = campaignId

	assert.NoError(t, addCampaign(c))
	assert.Equal(t, http.StatusCreated, c.Response().Status)
	assert.Equal(t, campaignId, rec.Body.String())
}

func TestUpdateCampaignMissingParamCampaign(t *testing.T) {
	c, rec, _ := setupMockContextCampaign("")

	assert.NoError(t, updateCampaign(c))
	assert.Equal(t, http.StatusBadRequest, c.Response().Status)
	assert.Equal(t, "invalid parameter campaignName: ", rec.Body.String())
}

func TestUpdateCampaignErrorReadingCampaignFromRequestBody(t *testing.T) {
	c, rec := setupMockContextCampaignWithBody(campaign, "")

	assert.EqualError(t, updateCampaign(c), "EOF")
	assert.Equal(t, 0, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

func TestUpdateCampaignError(t *testing.T) {
	c, rec, testCampaign := setupMockContextCampaign(campaign)

	mock := newMockDb(t)
	mock.updateCampaignParam = testCampaign
	forcedError := fmt.Errorf("forced scan error update campaign")
	mock.updateCampaignErr = forcedError

	assert.EqualError(t, updateCampaign(c), forcedError.Error())
	assert.Equal(t, 0, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

func TestUpdateCampaign(t *testing.T) {
	c, rec, testCampaign := setupMockContextCampaign(campaign)

	mock := newMockDb(t)
	mock.updateCampaignParam = testCampaign
	mock.updateCampaignGuid = campaignId

	assert.NoError(t, updateCampaign(c))
	assert.Equal(t, http.StatusOK, c.Response().Status)
	assert.Equal(t, campaignId, rec.Body.String())
}

func setupMockContextParticipant(participantJson string) (c echo.Context, rec *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest("", "/", strings.NewReader(participantJson))
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	return
}

func TestAddParticipantBodyInvalid(t *testing.T) {
	c, rec := setupMockContextParticipant("")

	assert.EqualError(t, addParticipant(c), "EOF")
	assert.Equal(t, 0, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

func TestAddParticipantCampaignMissing(t *testing.T) {
	participantJson := fmt.Sprintf(`{"campaignName":"%s", "loginName": "%s"}`, campaign, loginName)
	c, rec := setupMockContextParticipant(participantJson)

	mock := newMockDb(t)
	mock.insertParticipantPartier = &types.ParticipantStruct{
		CampaignName: campaign,
		LoginName:    loginName,
	}
	forcedError := fmt.Errorf("forced SQL insert error")
	mock.insertParticipantErr = forcedError

	assert.EqualError(t, addParticipant(c), forcedError.Error())
	assert.Equal(t, 0, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

func TestAddParticipant(t *testing.T) {
	participantJson := fmt.Sprintf(`{"campaignName":"%s", "scpName": "%s","loginName": "%s"}`, campaign, scpName, loginName)
	c, rec := setupMockContextParticipant(participantJson)

	mock := newMockDb(t)
	mock.insertParticipantPartier = &types.ParticipantStruct{
		CampaignName: campaign,
		ScpName:      scpName,
		LoginName:    loginName,
	}
	mock.insertParticipantGuid = participantID
	mock.insertParticipantJoinedAt = now

	assert.NoError(t, addParticipant(c))
	assert.Equal(t, http.StatusCreated, c.Response().Status)
	assert.True(t, strings.HasPrefix(rec.Body.String(), `{"guid":"`+participantID+`","endpoints":{"participantDetail"`), rec.Body.String())
	assert.True(t, strings.Contains(rec.Body.String(), `"loginName":"`+loginName+`"`), rec.Body.String())
}

func TestLogAddParticipantWithError(t *testing.T) {
	c, rec := setupMockContext()
	err := logAddParticipant(c)
	assert.EqualError(t, err, "EOF")
	assert.Equal(t, 0, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

func TestLogAddParticipantNoError(t *testing.T) {
	participantJson := fmt.Sprintf(`{"campaignName":"%s", "scpName": "%s","loginName": "%s"}`, campaign, scpName, loginName)
	c, rec := setupMockContextParticipant(participantJson)

	mock := newMockDb(t)
	mock.insertParticipantPartier = &types.ParticipantStruct{
		CampaignName: campaign,
		ScpName:      scpName,
		LoginName:    loginName,
	}
	mock.insertParticipantGuid = participantID
	mock.insertParticipantJoinedAt = now

	err := logAddParticipant(c)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusCreated, c.Response().Status)
	assert.True(t, strings.HasPrefix(rec.Body.String(), `{"guid":"`+participantID+`","endpoints":{"participantDetail"`), rec.Body.String())
	assert.True(t, strings.Contains(rec.Body.String(), `"loginName":"`+loginName+`"`), rec.Body.String())
}

func setupMockContextUpdateParticipant(participantJson string) (c echo.Context, rec *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest("", "/", strings.NewReader(participantJson))
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	return
}

func TestUpdateParticipantBodyInvalid(t *testing.T) {
	c, rec := setupMockContextUpdateParticipant("")

	assert.EqualError(t, updateParticipant(c), "EOF")
	assert.Equal(t, 0, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

// unit test values
const campaignId = "myCampaignId"
const campaign = "myCampaignName"
const scpName = "myScpName"
const participantID = "participantUUId"
const loginName = "loginName"
const teamName = "myTeamName"

func TestUpdateParticipantMissingParticipantID(t *testing.T) {
	participantJson := fmt.Sprintf(`{"loginName": "%s","campaignName": "%s", "scpName": "%s"}`, loginName, campaign, scpName)
	c, rec := setupMockContextUpdateParticipant(participantJson)

	mock := newMockDb(t)
	mock.updateParticipantPartier = &types.ParticipantStruct{
		CampaignName: campaign,
		ScpName:      scpName,
		LoginName:    loginName,
	}
	forcedError := fmt.Errorf("forced SQL insert error")
	mock.updateParticipantErr = forcedError

	assert.EqualError(t, updateParticipant(c), forcedError.Error())
	assert.Equal(t, 0, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

func TestUpdateParticipantUpdateError(t *testing.T) {
	participantJson := fmt.Sprintf(`{"guid": "%s","campaignName": "%s", "scpName": "%s", "loginName": "%s"}`, participantID, campaign, scpName, loginName)
	c, rec := setupMockContextUpdateParticipant(participantJson)

	mock := newMockDb(t)
	mock.updateParticipantPartier = &types.ParticipantStruct{
		ID:           participantID,
		CampaignName: campaign,
		ScpName:      scpName,
		LoginName:    loginName,
	}
	forcedError := fmt.Errorf("forced SQL insert error")
	mock.updateParticipantErr = forcedError

	assert.EqualError(t, updateParticipant(c), forcedError.Error())
	assert.Equal(t, 0, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

func TestUpdateParticipantNoRowsUpdated(t *testing.T) {
	participantJson := fmt.Sprintf(`{"guid": "%s", "campaignName": "%s", "scpName": "%s", "loginName": "%s", "teamName": "%s"}`, participantID, campaign, scpName, loginName, teamName)
	c, rec := setupMockContextUpdateParticipant(participantJson)

	mock := newMockDb(t)
	mock.updateParticipantPartier = &types.ParticipantStruct{
		ID:           participantID,
		CampaignName: campaign,
		ScpName:      scpName,
		LoginName:    loginName,
		TeamName:     teamName,
	}

	mock.updateScoreParticipant = &types.ParticipantStruct{ID: participantID}

	logger = zaptest.NewLogger(t)

	assert.NoError(t, updateParticipant(c))
	assert.Equal(t, http.StatusBadRequest, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

func TestUpdateParticipant(t *testing.T) {
	participantJson := fmt.Sprintf(`{"guid": "%s", "campaignName": "%s", "scpName": "%s", "loginName": "%s"}`, participantID, campaign, scpName, loginName)
	c, rec := setupMockContextUpdateParticipant(participantJson)

	mock := newMockDb(t)
	mock.updateParticipantPartier = &types.ParticipantStruct{
		ID:           participantID,
		CampaignName: campaign,
		ScpName:      scpName,
		LoginName:    loginName,
	}
	mock.updateParticipantRowsAffected = 1

	mock.updateScoreParticipant = &types.ParticipantStruct{
		ID: participantID,
	}

	assert.NoError(t, updateParticipant(c))
	assert.Equal(t, http.StatusNoContent, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

func setupMockContextTeam(teamJson string) (c echo.Context, rec *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest("", "/", strings.NewReader(teamJson))
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	return
}

func TestAddTeamMissingTeam(t *testing.T) {
	c, rec := setupMockContextTeam("")

	assert.EqualError(t, addTeam(c), "EOF")
	assert.Equal(t, 0, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

func TestAddTeamInsertError(t *testing.T) {
	teamName := "myTeamName"
	teamJson := `{"name": "` + teamName + `"}`
	c, rec := setupMockContextTeam(teamJson)

	mock := newMockDb(t)
	mock.insertTeamTm = &types.TeamStruct{
		Name: teamName,
	}
	forcedError := fmt.Errorf("forced SQL insert error")
	mock.insertTeamErr = forcedError

	assert.EqualError(t, addTeam(c), forcedError.Error())
	assert.Equal(t, 0, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

func TestAddTeam(t *testing.T) {
	teamJson := `{"campaignName": "` + campaign + `","name":"` + teamName + `"}`
	c, rec := setupMockContextTeam(teamJson)

	mock := newMockDb(t)
	mock.insertTeamTm = &types.TeamStruct{
		Name:         teamName,
		CampaignName: campaign,
	}

	teamID := "teamUUId"
	mock.insertTeamGuid = teamID

	assert.NoError(t, addTeam(c))
	assert.Equal(t, http.StatusCreated, c.Response().Status)
	assert.Equal(t, teamID, rec.Body.String())
}

func setupMockContextAddPersonToTeam(campaignName, scpName, loginName, teamName string) (c echo.Context, rec *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest("", "/", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.SetParamNames(ParamCampaignName, ParamScpName, ParamLoginName, ParamTeamName)
	c.SetParamValues(campaignName, scpName, loginName, teamName)
	return
}

func TestAddPersonToTeamMissingParameters(t *testing.T) {
	c, rec := setupMockContextAddPersonToTeam("", "", "", "")

	assert.NoError(t, addPersonToTeam(c))
	assert.Equal(t, http.StatusBadRequest, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

func TestAddPersonToTeamUpdateError(t *testing.T) {
	c, rec := setupMockContextAddPersonToTeam(campaign, scpName, loginName, teamName)

	mock := newMockDb(t)
	mock.updatePartTeamTeamName = teamName
	mock.updatePartTeamCampaignName = campaign
	mock.updatePartTeamSCPName = scpName
	mock.updatePartTeamLoginName = loginName
	forcedError := fmt.Errorf("forced SQL update error")
	mock.updatePartTeamErr = forcedError

	assert.EqualError(t, addPersonToTeam(c), forcedError.Error())
	assert.Equal(t, 0, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

func TestAddPersonToTeamZeroRowsAffected(t *testing.T) {
	c, rec := setupMockContextAddPersonToTeam(campaign, scpName, loginName, teamName)

	mock := newMockDb(t)
	mock.updatePartTeamCampaignName = campaign
	mock.updatePartTeamSCPName = scpName
	mock.updatePartTeamLoginName = loginName
	mock.updatePartTeamTeamName = teamName
	mock.updatePartTeamRowsAffected = 0

	assert.NoError(t, addPersonToTeam(c))
	assert.Equal(t, http.StatusBadRequest, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

func TestAddPersonToTeamSomeRowsAffected(t *testing.T) {
	c, rec := setupMockContextAddPersonToTeam(campaign, scpName, loginName, teamName)

	mock := newMockDb(t)
	mock.updatePartTeamCampaignName = campaign
	mock.updatePartTeamSCPName = scpName
	mock.updatePartTeamLoginName = loginName
	mock.updatePartTeamTeamName = teamName
	mock.updatePartTeamRowsAffected = 5

	assert.NoError(t, addPersonToTeam(c))
	assert.Equal(t, http.StatusNoContent, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

func setupMockContextParticipantDetail(campaignName, scpName, loginName string) (c echo.Context, rec *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest("", "/", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.SetParamNames(ParamCampaignName, ParamScpName, ParamLoginName)
	c.SetParamValues(campaignName, scpName, loginName)
	return
}

func TestGetParticipantDetailScanError(t *testing.T) {
	c, rec := setupMockContextParticipantDetail("", "", "")

	mock := newMockDb(t)
	forcedError := fmt.Errorf("forced Scan error")
	mock.selectPartDetailErr = forcedError

	assert.EqualError(t, getParticipantDetail(c), forcedError.Error())
	assert.Equal(t, 0, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

func TestGetParticipantDetail(t *testing.T) {
	c, rec := setupMockContextParticipantDetail(campaign, scpName, loginName)

	mock := newMockDb(t)
	mock.selectPartDetailCampName = campaign
	mock.selectPartDetailSCPName = scpName
	mock.selectPartDetailLoginName = loginName
	mock.selectPartDetailResult = &types.ParticipantStruct{
		ID:           participantID,
		CampaignName: campaign,
		ScpName:      scpName,
		LoginName:    loginName,
		JoinedAt:     now,
	}

	assert.NoError(t, getParticipantDetail(c))
	assert.Equal(t, http.StatusOK, c.Response().Status)
	assert.True(t, strings.HasPrefix(rec.Body.String(), `{"guid":"`+participantID+`","campaignName":"`+campaign+`","scpName":"`+scpName+`","loginName":"`+loginName+`"`), rec.Body.String())
}

func setupMockContextParticipantList(campaignName string) (c echo.Context, rec *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest("", "/", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.SetParamNames(ParamCampaignName)
	c.SetParamValues(campaignName)
	return
}

func TestGetParticipantsListError(t *testing.T) {
	campaignName := ""
	c, rec := setupMockContextParticipantList(campaignName)

	mock := newMockDb(t)
	forcedError := fmt.Errorf("forced Scan error")
	mock.selectPartInCampErr = forcedError

	assert.EqualError(t, getParticipantsList(c), forcedError.Error())
	assert.Equal(t, 0, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

func TestGetParticipantsList(t *testing.T) {
	c, rec := setupMockContextParticipantList(campaign)

	mock := newMockDb(t)
	mock.selectPartInCampCamp = campaign
	mock.selectPartInCampResult = []types.ParticipantStruct{
		{
			ID:           participantID,
			CampaignName: campaign,
			JoinedAt:     now,
		},
	}

	assert.NoError(t, getParticipantsList(c))
	assert.Equal(t, http.StatusOK, c.Response().Status)
	assert.True(t, strings.HasPrefix(rec.Body.String(), `[{"guid":"`+participantID+`","campaignName":"`+campaign+`","scpName":"","loginName":""`), rec.Body.String())
}

func TestValidateBug(t *testing.T) {
	_, _ = setupMockContext()
	logger = zaptest.NewLogger(t)
	assert.EqualError(t, validateBug(&types.BugStruct{}), "bug is not valid, empty campaign: bug: &{Id: Campaign: Category: PointValue:0}")
	assert.EqualError(t, validateBug(&types.BugStruct{Campaign: "myCampaign"}), "bug is not valid, empty category: bug: &{Id: Campaign:myCampaign Category: PointValue:0}")
	assert.EqualError(t, validateBug(&types.BugStruct{Campaign: "myCampaign", Category: ""}), "bug is not valid, empty category: bug: &{Id: Campaign:myCampaign Category: PointValue:0}")
	assert.EqualError(t, validateBug(&types.BugStruct{Campaign: "myCampaign", Category: "myCategory", PointValue: -1}), "bug is not valid, negative PointValue: bug: &{Id: Campaign:myCampaign Category:myCategory PointValue:-1}")
	assert.NoError(t, validateBug(&types.BugStruct{Campaign: "myCampaign", Category: "myCategory", PointValue: 0}))
}

func setupMockContextAddBug(bugJson string) (c echo.Context, rec *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest("", "/", strings.NewReader(bugJson))
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	return
}

func TestAddBugMissingBug(t *testing.T) {
	c, rec := setupMockContextAddBug("")

	assert.EqualError(t, addBug(c), "EOF")
	assert.Equal(t, 0, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

const category = "myCategory"

func TestAddBugScanError(t *testing.T) {
	c, rec := setupMockContextAddBug(`{"campaign": "` + campaign + `", "category":"` + category + `"}`)

	mock := newMockDb(t)
	mock.insertBugBug = &types.BugStruct{
		Campaign: campaign,
		Category: category,
	}
	forcedError := fmt.Errorf("forced insert bug error")
	mock.insertBugErr = forcedError

	assert.EqualError(t, addBug(c), forcedError.Error())
	assert.Equal(t, 0, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

func TestAddBugInvalidBug(t *testing.T) {
	c, rec := setupMockContextAddBug(`{}`)

	newMockDb(t)

	assert.EqualError(t, addBug(c), "bug is not valid, empty campaign: bug: &{Id: Campaign: Category: PointValue:0}")
	assert.Equal(t, 0, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}
func TestAddBug(t *testing.T) {
	pointValue := 9
	c, rec := setupMockContextAddBug(`{"campaign": "` + campaign + `", "category":"` + category + `","pointValue":` + strconv.Itoa(pointValue) + `}`)

	mock := newMockDb(t)
	mock.insertBugBug = &types.BugStruct{
		Campaign:   campaign,
		Category:   category,
		PointValue: pointValue,
	}
	bugId := "myBugId"
	mock.insertBugGuid = bugId

	assert.NoError(t, addBug(c))
	assert.Equal(t, http.StatusCreated, c.Response().Status)
	assert.True(t, strings.HasPrefix(rec.Body.String(), `{"guid":"`+bugId+`","endpoints":`), rec.Body.String())
	assert.True(t, strings.HasSuffix(rec.Body.String(), `"object":{"guid":"`+bugId+`","campaign":"`+campaign+`","category":"`+category+`","pointValue":`+strconv.Itoa(pointValue)+`}}`+"\n"), rec.Body.String())
}

func setupMockContextUpdateBug(campaign, bugCategory, pointValue string) (c echo.Context, rec *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest("", "/", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.SetParamNames(ParamCampaignName, ParamBugCategory, ParamPointValue)
	c.SetParamValues(campaign, bugCategory, pointValue)
	return
}

func TestUpdateBugInvalidPointValue(t *testing.T) {
	c, rec := setupMockContextUpdateBug("", "", "non-number")

	assert.EqualError(t, updateBug(c), `strconv.Atoi: parsing "non-number": invalid syntax`)
	assert.Equal(t, 0, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

func TestUpdateBugUpdateError(t *testing.T) {
	pointValue := 9
	c, rec := setupMockContextUpdateBug(campaign, category, strconv.Itoa(pointValue))

	mock := newMockDb(t)
	mock.updateBugBug = &types.BugStruct{
		Campaign:   campaign,
		Category:   category,
		PointValue: pointValue,
	}
	forcedError := fmt.Errorf("forced Update bug error")
	mock.updateBugErr = forcedError

	assert.EqualError(t, updateBug(c), forcedError.Error())
	assert.Equal(t, 0, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

func TestUpdateBugRowsAffectedZero(t *testing.T) {
	pointValue := 9
	c, rec := setupMockContextUpdateBug(campaign, category, strconv.Itoa(pointValue))

	mock := newMockDb(t)
	mock.updateBugBug = &types.BugStruct{
		Campaign:   campaign,
		Category:   category,
		PointValue: pointValue,
	}
	mock.updateBugRowsAffected = 0

	assert.NoError(t, updateBug(c))
	assert.Equal(t, http.StatusNotFound, c.Response().Status)
	assert.Equal(t, "Bug Category not found", rec.Body.String())
}

func TestUpdateBugInvalidBug(t *testing.T) {
	c, rec := setupMockContextUpdateBug("myCampaign", "myCategory", "-1")

	newMockDb(t)

	assert.EqualError(t, updateBug(c), "bug is not valid, negative PointValue: bug: &{Id: Campaign:myCampaign Category:myCategory PointValue:-1}")
	assert.Equal(t, 0, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

func TestUpdateBug(t *testing.T) {
	pointValue := 9
	c, rec := setupMockContextUpdateBug(campaign, category, strconv.Itoa(pointValue))

	mock := newMockDb(t)
	mock.updateBugBug = &types.BugStruct{
		Campaign:   campaign,
		Category:   category,
		PointValue: pointValue,
	}
	mock.updateBugRowsAffected = 5

	assert.NoError(t, updateBug(c))
	assert.Equal(t, http.StatusOK, c.Response().Status)
	assert.Equal(t, "Success", rec.Body.String())
}

func setupMockContextGetBugs() (c echo.Context, rec *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest("", "/", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	return
}

func TestGetBugsError(t *testing.T) {
	c, rec := setupMockContextGetBugs()

	mock := newMockDb(t)
	forcedError := fmt.Errorf("forced Select error")
	mock.selectBugsErr = forcedError

	assert.EqualError(t, getBugs(c), forcedError.Error())
	assert.Equal(t, 0, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

func TestGetBugs(t *testing.T) {
	c, rec := setupMockContextGetBugs()

	mock := newMockDb(t)
	bugId := "myBugId"
	category := "myCategory"
	pointValue := 9
	mock.selectBugsResult = []types.BugStruct{
		{
			Id:         bugId,
			Campaign:   campaign,
			Category:   category,
			PointValue: pointValue,
		},
	}

	assert.NoError(t, getBugs(c))
	assert.Equal(t, http.StatusOK, c.Response().Status)
	assert.Equal(t, `[{"guid":"`+bugId+`","campaign":"`+campaign+`","category":"`+category+`","pointValue":`+strconv.Itoa(pointValue)+`}]`+"\n", rec.Body.String())
}

func setupMockContextPutBugs(bugsJson string) (c echo.Context, rec *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest("", "/", strings.NewReader(bugsJson))
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	return
}

func TestPutBugsBodyInvalid(t *testing.T) {
	c, rec := setupMockContextPutBugs("")

	assert.EqualError(t, putBugs(c), "EOF")
	assert.Equal(t, 0, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

func TestPutBugsScanError(t *testing.T) {
	c, rec := setupMockContextPutBugs(
		`[{"campaign":"` + campaign + `","category":"` + category + `", "pointValue":5}]`)

	mock := newMockDb(t)
	mock.insertBugBug = &types.BugStruct{
		Campaign:   campaign,
		Category:   category,
		PointValue: 5,
	}
	forcedError := fmt.Errorf("forced Scan error")
	mock.insertBugErr = forcedError

	assert.EqualError(t, putBugs(c), forcedError.Error())
	assert.Equal(t, 0, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

func TestPutBugsOneBugInvalidBug(t *testing.T) {
	c, rec := setupMockContextPutBugs(`[{}]`)

	newMockDb(t)

	assert.EqualError(t, putBugs(c), "bug is not valid, empty campaign: bug: &{Id: Campaign: Category: PointValue:0}")
	assert.Equal(t, 0, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}
func TestPutBugsOneBug(t *testing.T) {
	c, rec := setupMockContextPutBugs(`[{"campaign":"myCampaign","category":"bugCat2", "pointValue":5}]`)

	mock := newMockDb(t)
	bugId := "myBugId"
	mock.insertBugBug = &types.BugStruct{
		Campaign:   "myCampaign",
		Category:   "bugCat2",
		PointValue: 5,
	}
	mock.insertBugGuid = bugId

	assert.NoError(t, putBugs(c))
	assert.Equal(t, http.StatusCreated, c.Response().Status)
	assert.Equal(t, `{"guid":"`+bugId+`","endpoints":null,"object":[{"guid":"`+bugId+`","campaign":"myCampaign","category":"bugCat2","pointValue":5}]}`+"\n", rec.Body.String())
}

func TestPutBugsMultipleBugs(t *testing.T) {
	c, rec := setupMockContextPutBugs(`[{"campaign":"myCampaign","category":"bugCat2", "pointValue":5}, {"campaign":"myCampaign","category":"bugCat3", "pointValue":9}]`)

	mock := newMockDb(t)
	mock.insertBugBug = &types.BugStruct{
		Campaign:   "myCampaign",
		Category:   "bugCat2",
		PointValue: 5,
	}
	bugId := "myBugId"
	mock.insertBugGuid = bugId
	bugId2 := fmt.Sprintf("%s%d", bugId, 1)

	assert.NoError(t, putBugs(c))
	assert.Equal(t, http.StatusCreated, c.Response().Status)
	assert.Equal(t, `{"guid":"`+bugId+`","endpoints":null,"object":[{"guid":"`+bugId+`","campaign":"myCampaign","category":"bugCat2","pointValue":5},{"guid":"`+bugId2+`","campaign":"myCampaign","category":"bugCat3","pointValue":9}]}`+"\n", rec.Body.String())
}

func setupMockContextParticipantDelete(campaignName, scpName, loginName string) (c echo.Context, rec *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.SetParamNames(ParamCampaignName, ParamScpName, ParamLoginName)
	c.SetParamValues(campaignName, scpName, loginName)
	return
}

func TestDeleteParticipant(t *testing.T) {
	c, rec := setupMockContextParticipantDelete(campaign, scpName, loginName)

	mock := newMockDb(t)
	mock.deletePartCampaign = campaign
	mock.deletePartSCPName = scpName
	mock.deletePartLoginName = loginName
	mock.deletePartGuid = participantID

	assert.NoError(t, deleteParticipant(c))
	assert.Equal(t, http.StatusOK, c.Response().Status)
	assert.Equal(t, fmt.Sprintf("\"deleted participant: campaign: %s, scpName: %s, loginName: %s, participant.id: %s\"\n", campaign, scpName, loginName, participantID), rec.Body.String())
}

func TestDeleteParticipantWithDBDeleteError(t *testing.T) {
	c, rec := setupMockContextParticipantDelete(campaign, scpName, loginName)

	mock := newMockDb(t)
	mock.deletePartCampaign = campaign
	mock.deletePartSCPName = scpName
	mock.deletePartLoginName = loginName
	forcedError := fmt.Errorf("forced delete error")
	mock.deletePartErr = forcedError

	assert.EqualError(t, deleteParticipant(c), forcedError.Error())
	assert.Equal(t, 0, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

func TestValidScoreErrorValidatingOrganization(t *testing.T) {
	_, _ = setupMockContext()

	mock := newMockDb(t)
	msg := &types.ScoringMessage{EventSource: db.TestEventSourceValid, RepoOwner: db.TestOrgValid}
	mock.validOrgParam = msg
	forcedError := fmt.Errorf("forced org exists query error")
	mock.validOrgErr = forcedError

	activeParticipantsToScore, err := validScore(msg, now)
	assert.EqualError(t, err, forcedError.Error())
	assert.Equal(t, 0, len(activeParticipantsToScore))
}

func TestValidScoreOrganizationNotValid(t *testing.T) {
	_, _ = setupMockContext()

	mock := newMockDb(t)
	msg := &types.ScoringMessage{EventSource: db.TestEventSourceValid, RepoOwner: db.TestOrgValid}
	mock.validOrgParam = msg
	mock.validOrgResult = false

	activeParticipantsToScore, err := validScore(msg, now)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(activeParticipantsToScore))
}

func TestValidScoreUnknownRepoOwner(t *testing.T) {
	_, _ = setupMockContext()

	mock := newMockDb(t)
	msg := &types.ScoringMessage{EventSource: db.TestEventSourceValid, RepoOwner: db.TestOrgValid}
	mock.validOrgParam = msg
	mock.validOrgResult = false

	activeParticipantsToScore, err := validScore(msg, now)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(activeParticipantsToScore))
}

func setupMockContext() (c echo.Context, rec *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	return
}

func setupMockContextWithBody(method string, body string) (c echo.Context, rec *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, "/", strings.NewReader(body))
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	return
}

func TestValidScoreParticipantNotRegistered(t *testing.T) {
	mock := newMockDb(t)
	msg := types.ScoringMessage{EventSource: db.TestEventSourceValid, RepoOwner: db.TestOrgValid, TriggerUser: "unregisteredUser"}
	mock.validOrgParam = &msg

	_, _ = setupMockContext()

	activeParticipantsToScore, err := validScore(&msg, now)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(activeParticipantsToScore))
}

func TestValidScoreParticipantError(t *testing.T) {
	mock := newMockDb(t)
	msg := &types.ScoringMessage{EventSource: db.TestEventSourceValid, RepoOwner: db.TestOrgValid, TriggerUser: loginName}
	mock.validOrgParam = msg
	forcedError := fmt.Errorf("forced current campaign read error")
	mock.validOrgErr = forcedError

	_, _ = setupMockContext()

	activeParticipantsToScore, err := validScore(msg, now)
	assert.EqualError(t, err, forcedError.Error())
	assert.Equal(t, 0, len(activeParticipantsToScore))
}

func TestValidScoreParticipantErrorReadingParticipant(t *testing.T) {
	mock := newMockDb(t)
	setupMockDBOrgValid(mock)
	msg := &types.ScoringMessage{EventSource: db.TestEventSourceValid, RepoOwner: db.TestOrgValid, TriggerUser: loginName}
	mock.validOrgParam = msg
	mock.partiesToScoreMsg = msg
	mock.partiesToScoreNow = now

	forcedError := fmt.Errorf("forced current campaign read error")
	mock.partiesToScoreErr = forcedError

	_, _ = setupMockContext()

	activeParticipantsToScore, err := validScore(msg, now)
	assert.EqualError(t, err, forcedError.Error())
	assert.Equal(t, 0, len(activeParticipantsToScore))
}

func TestValidScoreParticipant(t *testing.T) {
	mock := newMockDb(t)
	setupMockDBOrgValid(mock)
	msg := &types.ScoringMessage{EventSource: db.TestEventSourceValid, RepoOwner: db.TestOrgValid, TriggerUser: loginName}
	mock.validOrgParam = msg
	mock.partiesToScoreMsg = msg
	mock.partiesToScoreNow = now
	mock.partiesToScoreResult = []types.ParticipantStruct{
		{
			ID:           "someId",
			CampaignName: "someCampaign",
			ScpName:      "someSCP",
			LoginName:    "someLoginName",
		},
	}

	_, _ = setupMockContext()

	activeParticipantsToScore, err := validScore(msg, now)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(activeParticipantsToScore))
	assert.Equal(t, "someCampaign", activeParticipantsToScore[0].CampaignName)
	assert.Equal(t, "someSCP", activeParticipantsToScore[0].ScpName)
}

func setupMockDBOrgValid(mock *MockBBashDB) {
	mock.validOrgParam = &types.ScoringMessage{EventSource: db.TestEventSourceValid, RepoOwner: db.TestOrgValid}
	mock.validOrgResult = true
}

func TestTraverseBugCountsEmpty(t *testing.T) {
	points := float64(1)
	scored := float64(2)
	bugCounts := map[string]interface{}{}

	err := traverseBugCounts(nil, "", &points, &scored, &bugCounts)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), points)
	assert.Equal(t, float64(2), scored)
}

func TestTraverseBugCountsSimple(t *testing.T) {
	bugType := "myBugType"

	mock := newMockDb(t)
	mock.selectPointValueBugType = bugType
	mock.selectPointValueResult = 2

	points := float64(1)
	scored := float64(2)
	bugCounts := map[string]interface{}{
		bugType: float64(3),
	}

	err := traverseBugCounts(nil, "", &points, &scored, &bugCounts)
	assert.NoError(t, err)
	assert.Equal(t, float64(7), points)
	assert.Equal(t, float64(5), scored)
}

func TestTraverseBugCountsNestedMap(t *testing.T) {
	bugType := "myBugType"
	nestedBugType := "myNestedBugType"

	mock := newMockDb(t)
	mock.selectPointValueBugType = nestedBugType
	mock.selectPointValueResult = 2

	points := float64(1)
	scored := float64(2)
	mapNestedBugType := map[string]interface{}{
		nestedBugType: float64(3),
	}
	bugCounts := map[string]interface{}{
		bugType: mapNestedBugType,
	}

	err := traverseBugCounts(nil, "", &points, &scored, &bugCounts)
	assert.NoError(t, err)
	assert.Equal(t, float64(7), points)
	assert.Equal(t, float64(5), scored)
}

func TestTraverseBugCountsSimpleAndNestedMap(t *testing.T) {
	bugType := "myBugType"
	nestedBugType := "myNestedBugType"

	mock := newMockDb(t)
	mock.assertParameters = false
	mock.selectPointValueResult = 2

	points := float64(1)
	scored := float64(2)
	mapNestedBugType := map[string]interface{}{
		nestedBugType: float64(3),
	}
	bugCounts := map[string]interface{}{
		"bugTypeSimpleFirst": float64(2),
		bugType:              mapNestedBugType,
		"bugTypeSimpleLast":  float64(4),
	}

	err := traverseBugCounts(nil, "", &points, &scored, &bugCounts)
	assert.NoError(t, err)
	assert.Equal(t, float64(19), points)
	assert.Equal(t, float64(11), scored)
}

func TestTraverseBugCountsSimpleAndNestedMapNonClassified(t *testing.T) {
	bugType := "myBugType"
	nestedBugType := "myNestedBugType"

	mock := newMockDb(t)
	mock.assertParameters = false

	points := float64(1)
	scored := float64(2)
	mapNestedBugType := map[string]interface{}{
		nestedBugType: float64(3),
	}
	bugCounts := map[string]interface{}{
		"bugTypeSimpleFirst": float64(2),
		bugType:              mapNestedBugType,
		"bugTypeSimpleLast":  float64(4),
	}

	err := traverseBugCounts(nil, "", &points, &scored, &bugCounts)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), points)
	assert.Equal(t, float64(11), scored)
}

func TestScorePointsNothing(t *testing.T) {
	msg := &types.ScoringMessage{}
	points := scorePoints(msg, campaign)
	assert.Equal(t, float64(0), points)
}

func TestScorePoints(t *testing.T) {
	mock := newMockDb(t)
	msg := &types.ScoringMessage{BugCounts: map[string]interface{}{"myBugType": float64(1)}}
	mock.selectPointValueMsg = msg
	mock.selectPointValueCampaign = campaign
	mock.selectPointValueBugType = "myBugType"
	mock.selectPointValueResult = 1

	_, _ = setupMockContext()

	points := scorePoints(msg, campaign)
	assert.Equal(t, float64(1), points)
}

func TestScorePointsWithTraverseError(t *testing.T) {
	mock := newMockDb(t)
	msg := &types.ScoringMessage{BugCounts: map[string]interface{}{
		"myBadBugType": "bogusValueType",
		"myGoodugType": float64(2),
	}}
	mock.assertParameters = false
	mock.selectPointValueResult = 2

	_, _ = setupMockContext()

	points := scorePoints(msg, campaign)
	assert.Equal(t, float64(4), points)
}

func TestScorePointsFixedTwoThreePointers(t *testing.T) {
	mock := newMockDb(t)
	mock.selectPointValueResult = 3
	bugType := "threePointBugType"
	msg := &types.ScoringMessage{BugCounts: map[string]interface{}{bugType: float64(2)}}
	mock.selectPointValueMsg = msg
	mock.selectPointValueCampaign = campaign
	mock.selectPointValueBugType = bugType

	points := scorePoints(msg, campaign)
	assert.Equal(t, float64(6), points)
}

func TestScorePointsWithOptMap(t *testing.T) {
	mock := newMockDb(t)
	mock.assertParameters = false
	mock.selectPointValueResult = 3

	// similar to this:
	// "fixed-bug-types":{"opt":{"semgrep":{"node_password":1,"node_username":1}}}
	mapSemGroupBugType := map[string]interface{}{"sprintf-host-port": float64(2)}
	mapSemGrep := map[string]interface{}{"semgrep": mapSemGroupBugType}
	mapBugTypes := map[string]interface{}{
		"G104":       float64(1),
		"ShellCheck": float64(1),
		"opt":        mapSemGrep,
	}
	msg := types.ScoringMessage{
		BugCounts: mapBugTypes,
	}

	points := scorePoints(&msg, campaign)
	assert.Equal(t, float64(12), points)
}

func TestScorePointsBonusForNonClassified(t *testing.T) {
	msg := &types.ScoringMessage{TotalFixed: 1}
	points := scorePoints(msg, campaign)
	assert.Equal(t, float64(1), points)
}

func TestProcessScoringMessageInvalidScore_Error(t *testing.T) {
	msg := types.ScoringMessage{EventSource: db.TestEventSourceValid, RepoOwner: db.TestOrgValid, TriggerUser: loginName}

	mock := newMockDb(t)
	setupMockDBOrgValid(mock)
	msgLowerCase := msg
	msgLowerCase.TriggerUser = strings.ToLower(msgLowerCase.TriggerUser)
	mock.validOrgParam = &msgLowerCase
	forcedError := fmt.Errorf("forced validScore error")
	mock.validOrgErr = forcedError

	err := processScoringMessage(mock, now, &msg)
	assert.EqualError(t, err, forcedError.Error())
}

func TestProcessScoringMessageInvalidScore_NoTriggerUserFound(t *testing.T) {
	msg := &types.ScoringMessage{EventSource: db.TestEventSourceValid, RepoOwner: db.TestOrgValid, TriggerUser: loginName}

	mock := newMockDb(t)
	setupMockDBOrgValid(mock)
	msgLowerCase := msg
	msgLowerCase.TriggerUser = strings.ToLower(loginName)
	mock.validOrgParam = msgLowerCase
	mock.partiesToScoreMsg = msgLowerCase
	// caller users Time.now(), so don't assert time parameter
	mock.partiesToScoreNowSkip = true

	err := processScoringMessage(mock, now, msg)
	assert.NoError(t, err)
}

func TestProcessScoringMessageUserCapitalizationMismatch(t *testing.T) {
	loginNameWithCaps := "MYGithubName"
	//loginNameLowerCase := strings.ToLower(loginName)
	repoName := "myRepoName"
	prId := -5
	msg := &types.ScoringMessage{EventSource: db.TestEventSourceValid, RepoOwner: db.TestOrgValid, TriggerUser: loginNameWithCaps, RepoName: repoName, PullRequest: prId}

	mock := newMockDb(t)
	setupMockDBOrgValid(mock)
	msgLowerCase := msg
	msgLowerCase.TriggerUser = strings.ToLower(loginNameWithCaps)
	mock.validOrgParam = msgLowerCase
	mock.partiesToScoreMsg = msgLowerCase
	// caller users Time.now(), so don't assert time parameter
	mock.partiesToScoreNowSkip = true

	err := processScoringMessage(mock, now, msg)
	assert.NoError(t, err)
}

func TestProcessScoringMessageOne(t *testing.T) {
	repoName := "myRepoName"
	prId := -5
	msg := &types.ScoringMessage{EventSource: db.TestEventSourceValid, RepoOwner: db.TestOrgValid, TriggerUser: loginName, RepoName: repoName, PullRequest: prId}

	mock := newMockDb(t)
	setupMockDBOrgValid(mock)
	msgLowerCase := msg
	msgLowerCase.TriggerUser = strings.ToLower(loginName)
	mock.validOrgParam = msgLowerCase
	mock.partiesToScoreMsg = msgLowerCase
	// caller users Time.now(), so don't assert time parameter
	mock.partiesToScoreNowSkip = true

	err := processScoringMessage(mock, now, msg)
	assert.NoError(t, err)
}

func TestProcessScoringMessageTwoParticipantsToScore(t *testing.T) {
	repoName := "myRepoName"
	prId := -5
	msg := &types.ScoringMessage{EventSource: db.TestEventSourceValid, RepoOwner: db.TestOrgValid, TriggerUser: loginName,
		RepoName: repoName, PullRequest: prId, TotalFixed: 2}

	mock := newMockDb(t)
	setupMockDBOrgValid(mock)
	msgLowerCase := msg
	msgLowerCase.TriggerUser = strings.ToLower(loginName)
	mock.validOrgParam = msgLowerCase
	mock.partiesToScoreMsg = msgLowerCase
	// caller users Time.now(), so don't assert time parameter
	mock.partiesToScoreNowSkip = true
	mock.partiesToScoreResult = []types.ParticipantStruct{
		{
			ID:           "someId",
			CampaignName: "someCampaign",
			ScpName:      "someSCP",
			LoginName:    "someLoginName",
		},
		{
			ID:           "someId2",
			CampaignName: "someCampaign2",
			ScpName:      "someSCP2",
			LoginName:    "someLoginName2",
		},
	}
	mock.priorScoreResult = 4

	mock.priorScoreParticipant = &mock.partiesToScoreResult[0]
	mock.priorScoreMsg = msg

	mock.insertScoreEvtPartier = &mock.partiesToScoreResult[0]
	mock.insertScoreEvtMsg = msg
	mock.insertScoreEvtNewPoints = 2

	err := processScoringMessage(mock, now, msg)
	assert.NoError(t, err)
	assert.Equal(t, float64(-3), updateScoreLastDelta)
}

func TestProcessScoringMessageParticipantPriorScoreError(t *testing.T) {
	repoName := "myRepoName"
	prId := -5
	msg := &types.ScoringMessage{EventSource: db.TestEventSourceValid, RepoOwner: db.TestOrgValid, TriggerUser: loginName, RepoName: repoName, PullRequest: prId}

	mock := newMockDb(t)
	setupMockDBOrgValid(mock)
	msgLowerCase := msg
	msgLowerCase.TriggerUser = strings.ToLower(loginName)
	mock.validOrgParam = msgLowerCase
	mock.partiesToScoreMsg = msgLowerCase
	// caller users Time.now(), so don't assert time parameter
	mock.partiesToScoreNowSkip = true
	mock.partiesToScoreResult = []types.ParticipantStruct{
		{
			ID:           "someId",
			CampaignName: "someCampaign",
			ScpName:      "someSCP",
			LoginName:    "someLoginName",
		},
	}

	mock.priorScoreParticipant = &mock.partiesToScoreResult[0]
	mock.priorScoreMsg = msgLowerCase

	mock.insertScoreEvtPartier = &mock.partiesToScoreResult[0]
	mock.insertScoreEvtMsg = msgLowerCase
	forcedError := fmt.Errorf("forced prior score error")
	mock.insertScoreEvtErr = forcedError

	err := processScoringMessage(mock, now, msg)
	assert.EqualError(t, err, forcedError.Error())
}

func TestProcessScoringMessageParticipantUpdateScoreError(t *testing.T) {
	repoName := "myRepoName"
	prId := -5
	msg := &types.ScoringMessage{EventSource: db.TestEventSourceValid, RepoOwner: db.TestOrgValid, TriggerUser: loginName, RepoName: repoName, PullRequest: prId}

	mock := newMockDb(t)
	setupMockDBOrgValid(mock)
	msgLowerCase := msg
	msgLowerCase.TriggerUser = strings.ToLower(loginName)
	mock.validOrgParam = msgLowerCase
	mock.partiesToScoreMsg = msgLowerCase
	// caller users Time.now(), so don't assert time parameter
	mock.partiesToScoreNowSkip = true
	mock.partiesToScoreResult = []types.ParticipantStruct{
		{
			ID:           "someId",
			CampaignName: "someCampaign",
			ScpName:      "someSCP",
			LoginName:    "someLoginName",
		},
	}

	mock.priorScoreParticipant = &mock.partiesToScoreResult[0]
	mock.priorScoreMsg = msgLowerCase

	mock.insertScoreEvtPartier = &mock.partiesToScoreResult[0]
	mock.insertScoreEvtMsg = msgLowerCase

	mock.updateScoreParticipant = &mock.partiesToScoreResult[0]
	forcedError := fmt.Errorf("forced update participant score error")
	mock.updateScoreErr = forcedError

	err := processScoringMessage(mock, now, msg)
	assert.EqualError(t, err, forcedError.Error())
}

func TestProcessScoringMessageParticipant(t *testing.T) {
	repoName := "myRepoName"
	prId := -5
	msg := &types.ScoringMessage{EventSource: db.TestEventSourceValid, RepoOwner: db.TestOrgValid, TriggerUser: loginName, RepoName: repoName, PullRequest: prId,
		BugCounts: map[string]interface{}{category: float64(2)}}

	mock := newMockDb(t)
	setupMockDBOrgValid(mock)
	msgLowerCase := msg
	msgLowerCase.TriggerUser = strings.ToLower(loginName)
	mock.validOrgParam = msgLowerCase
	mock.partiesToScoreMsg = msgLowerCase
	// caller users Time.now(), so don't assert time parameter
	mock.partiesToScoreNowSkip = true
	mock.partiesToScoreResult = []types.ParticipantStruct{
		{
			ID:           "someId",
			CampaignName: campaign,
			ScpName:      "someSCP",
			LoginName:    "someLoginName",
		},
	}

	mock.selectPointValueMsg = msgLowerCase
	mock.selectPointValueCampaign = campaign
	mock.selectPointValueBugType = category
	mock.selectPointValueResult = 3

	mock.priorScoreParticipant = &mock.partiesToScoreResult[0]
	mock.priorScoreMsg = msgLowerCase
	mock.priorScoreResult = 2

	mock.insertScoreEvtPartier = &mock.partiesToScoreResult[0]
	mock.insertScoreEvtMsg = msgLowerCase
	mock.insertScoreEvtNewPoints = 6

	mock.updateScoreParticipant = &mock.partiesToScoreResult[0]
	mock.updateScoreDelta = 4

	err := processScoringMessage(mock, now, msg)
	assert.NoError(t, err)
}

func TestGetSourceControlProvidersQueryError(t *testing.T) {
	c, rec := setupMockContext()

	mock := newMockDb(t)
	forcedError := fmt.Errorf("forced scp error")
	mock.getSCPPsErr = forcedError

	err := getSourceControlProviders(c)
	assert.EqualError(t, err, forcedError.Error())
	assert.Equal(t, 0, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

func TestGetSourceControlProviders(t *testing.T) {
	c, rec := setupMockContext()

	mock := newMockDb(t)
	mock.getSCPPs = []types.SourceControlProviderStruct{
		{
			ID:      "someId",
			SCPName: "someSCP",
			Url:     "someUrl",
		},
	}

	err := getSourceControlProviders(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, c.Response().Status)
	assert.Equal(t, "[{\"guid\":\"someId\",\"scpName\":\"someSCP\",\"url\":\"someUrl\"}]\n", rec.Body.String())
}

func TestGetOrganizationsError(t *testing.T) {
	c, rec := setupMockContext()

	mock := newMockDb(t)
	forcedErr := fmt.Errorf("forced org list error")
	mock.getOrganizationsErr = forcedErr

	err := getOrganizations(c)
	assert.EqualError(t, err, forcedErr.Error())
	assert.Equal(t, 0, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

func TestGetOrganizations(t *testing.T) {
	c, rec := setupMockContext()

	mock := newMockDb(t)
	mock.getOrganizationsResult = []types.OrganizationStruct{
		{
			ID:           "someId",
			SCPName:      "someSCP",
			Organization: "someOrg",
		},
	}

	err := getOrganizations(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, c.Response().Status)
	assert.Equal(t, "[{\"guid\":\"someId\",\"scpName\":\"someSCP\",\"organization\":\"someOrg\"}]\n", rec.Body.String())
}

func TestAddOrganizationBodyBad(t *testing.T) {
	c, rec := setupMockContext()

	err := addOrganization(c)
	assert.EqualError(t, err, "EOF")
	assert.Equal(t, 0, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

func TestAddOrganizationInsertError(t *testing.T) {
	c, rec := setupMockContextWithBody(http.MethodPut, "{\"organization\":\"myOrganizationName\"}")

	mock := newMockDb(t)
	mock.insertOrganizationParam = &types.OrganizationStruct{
		Organization: "myOrganizationName",
	}
	forcedError := fmt.Errorf("forced org add error")
	mock.insertOrganizationErr = forcedError

	err := addOrganization(c)
	assert.EqualError(t, err, forcedError.Error())
	assert.Equal(t, 0, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

func TestAddOrganization(t *testing.T) {
	c, rec := setupMockContextWithBody(http.MethodPut, "{\"organization\":\"myOrganizationName\"}")

	mock := newMockDb(t)
	mock.insertOrganizationParam = &types.OrganizationStruct{
		Organization: "myOrganizationName",
	}
	mock.insertOrganizationGuid = "someId"

	err := addOrganization(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, c.Response().Status)
	assert.Equal(t, "someId", rec.Body.String())
}

func TestDeleteOrganizationDeleteError(t *testing.T) {
	c, rec := setupMockContext()

	mock := newMockDb(t)

	forcedError := fmt.Errorf("forced org delete error")
	mock.deleteOrgErr = forcedError

	err := deleteOrganization(c)
	assert.EqualError(t, err, forcedError.Error())
	assert.Equal(t, 0, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

func TestDeleteOrganizationNotFound(t *testing.T) {
	c, rec := setupMockContext()

	mock := newMockDb(t)
	mock.deleteOrgRowsAffected = 0

	err := deleteOrganization(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, c.Response().Status)
	assert.Equal(t, "\"no organization: scpName: , name: \"\n", rec.Body.String())
}

func TestDeleteOrganization(t *testing.T) {
	c, rec := setupMockContext()

	mock := newMockDb(t)
	mock.deleteOrgRowsAffected = 1

	err := deleteOrganization(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, c.Response().Status)
	assert.Equal(t, "", rec.Body.String())
}

func saveEnvAdminCredentials(t *testing.T) (resetInfoCreds func()) {
	origInfoUsername := os.Getenv(envAdminUsername)
	origInfoPassword := os.Getenv(envAdminPassword)
	resetInfoCreds = func() {
		resetEnvVariable(t, envAdminUsername, origInfoUsername)
		resetEnvVariable(t, envAdminUsername, origInfoPassword)
	}

	// setup testing logger while we're here
	logger = zaptest.NewLogger(t)

	return
}

func TestInfoBasicValidatorMissingEnv(t *testing.T) {
	resetInfoCreds := saveEnvAdminCredentials(t)
	defer resetInfoCreds()
	assert.NoError(t, os.Unsetenv(envAdminUsername))
	assert.NoError(t, os.Unsetenv(envAdminPassword))

	isValid, err := infoBasicValidator("yadda", "bing", nil)
	assert.NoError(t, err)
	assert.False(t, isValid)
}

func TestInfoBasicValidatorInValid(t *testing.T) {
	resetInfoCreds := saveEnvAdminCredentials(t)
	defer resetInfoCreds()
	assert.NoError(t, os.Setenv(envAdminUsername, "yadda"))
	assert.NoError(t, os.Setenv(envAdminPassword, "Doh!"))

	isValid, err := infoBasicValidator("yadda", "bing", nil)
	assert.NoError(t, err)
	assert.False(t, isValid)
}

func TestInfoBasicValidatorValid(t *testing.T) {
	resetInfoCreds := saveEnvAdminCredentials(t)
	defer resetInfoCreds()
	assert.NoError(t, os.Setenv(envAdminUsername, "yadda"))
	assert.NoError(t, os.Setenv(envAdminPassword, "bing"))

	isValid, err := infoBasicValidator("yadda", "bing", nil)
	assert.NoError(t, err)
	assert.True(t, isValid)
}

func TestLogTelemetry(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	q := req.URL.Query()
	q.Add(qpFeature, "testFeature")
	q.Add(qpCall, "testCaller")
	req.URL.RawQuery = q.Encode()

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	logger = zaptest.NewLogger(t)
	logTelemetry(c)
}

func TestLogTelemetryNoQueryParameters(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	logger = zaptest.NewLogger(t)
	logTelemetry(c)
}

func TestProcessScoringMessage(t *testing.T) {
	mock := newMockDb(t)
	setupMockDBOrgValid(mock)

	mock.assertParameters = false
	now := time.Now()
	// caller users Time.now(), so don't assert time parameter
	//mock.partiesToScoreNowSkip = true
	mock.partiesToScoreNow = now
	mock.partiesToScoreResult = []types.ParticipantStruct{
		{
			ID:           "someId",
			CampaignName: "someCampaign",
			ScpName:      "someSCP",
			LoginName:    "someLoginName",
		},
	}

	mapSprintf := map[string]interface{}{
		"sprintf-host-port": float64(2),
	}
	mapSemGrep := map[string]interface{}{
		"semgrep": mapSprintf,
	}
	mapBugTypes := map[string]interface{}{
		"opt":        mapSemGrep,
		"G104":       float64(1),
		"ShellCheck": float64(1),
	}

	msg := &types.ScoringMessage{
		BugCounts: mapBugTypes,
	}
	err := processScoringMessage(mock, now, msg)
	assert.NoError(t, err)
}

func TestBeginLogPolling(t *testing.T) {
	logger = zaptest.NewLogger(t)

	_, sqlDb, closeDbFunc := db.SetupMockDB(t)
	defer closeDbFunc()
	// side effect: set up the postgresDB var
	scoreDB = sqlDb

	quit, errChan, err := beginLogPolling()
	defer func() {
		//close(quit)
		//close(errChan)
	}()
	assert.NoError(t, err)
	assert.NotNil(t, quit)
	assert.NotNil(t, errChan)
}

func closePollIfSet() {
	if stopPoll != nil {
		close(stopPoll)
	}
}
func TestRestartPolling(t *testing.T) {
	logger = zaptest.NewLogger(t)

	_, sqlDb, closeDbFunc := db.SetupMockDB(t)
	defer closeDbFunc()
	// side effect: set up the postgresDB var
	scoreDB = sqlDb

	// fake stopPolling chan
	closePollIfSet()
	stopPoll = make(chan bool)

	err := restartPolling(nil)
	assert.NoError(t, err)
	assert.NotNil(t, stopPoll)
}

func TestStopPolling(t *testing.T) {
	// fake stopPolling chan
	closePollIfSet()
	stopPoll = make(chan bool)

	assert.NoError(t, stopPolling(nil))
	assert.Nil(t, stopPoll)
	// allow time for poll channel to finish logging during shutdown
	time.Sleep(1 * time.Second)
}

func TestSetPollDateEmptyBody(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest("", "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	assert.EqualError(t, setPollDate(c), "EOF")
}

func setupMockContextPollDate(t *testing.T, poll types.Poll) echo.Context {
	logger = zaptest.NewLogger(t)

	e := echo.New()

	bodyBytes, err := json.Marshal(poll)
	assert.NoError(t, err)
	req := httptest.NewRequest("", "/", strings.NewReader(string(bodyBytes)))
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return c
}

func TestSetPollDateSelectError(t *testing.T) {
	c := setupMockContextPollDate(t, types.Poll{})

	mock, dbFake, closeDbFunc := db.SetupMockDB(t)
	defer closeDbFunc()
	// side effect: set up the postgresDB var
	scoreDB = dbFake
	pollDB = db.NewDBPoll(scoreDB.GetDb(), logger)

	forcedError := fmt.Errorf("forced select poll error")
	db.SetupMockPollSelectForcedError(mock, forcedError, "1")

	assert.EqualError(t, setPollDate(c), forcedError.Error())
}

func TestSetPollDateUpdateError(t *testing.T) {
	c := setupMockContextPollDate(t, types.Poll{})

	mock, dbFake, closeDbFunc := db.SetupMockDB(t)
	defer closeDbFunc()
	// side effect: set up the postgresDB var
	scoreDB = dbFake
	pollDB = db.NewDBPoll(scoreDB.GetDb(), logger)

	db.SetupMockPollSelectAndUpdate(mock, "1", now, 1)

	err := setPollDate(c)
	assert.True(t, strings.HasPrefix(err.Error(), "ExecQuery 'UPDATE poll"))
}

func TestSetPollDate(t *testing.T) {
	c := setupMockContextPollDate(t, types.Poll{})

	mock, dbFake, closeDbFunc := db.SetupMockDB(t)
	defer closeDbFunc()
	// side effect: set up the postgresDB var
	scoreDB = dbFake
	pollDB = db.NewDBPoll(scoreDB.GetDb(), logger)

	db.SetupMockPollSelectAndUpdateAnyUpdateTime(mock, "1", now, 1)

	err := setPollDate(c)
	assert.NoError(t, err)
}
