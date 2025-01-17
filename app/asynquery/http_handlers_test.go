package asynquery

import (
	"bytes"
	"crypto/ecdsa"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/OdyseeTeam/odysee-api/app/auth"
	"github.com/OdyseeTeam/odysee-api/app/query"
	"github.com/OdyseeTeam/odysee-api/app/wallet"
	"github.com/OdyseeTeam/odysee-api/internal/e2etest"
	"github.com/OdyseeTeam/odysee-api/internal/test"
	"github.com/OdyseeTeam/odysee-api/internal/testdeps"
	"github.com/OdyseeTeam/odysee-api/models"
	"github.com/OdyseeTeam/odysee-api/pkg/keybox"
	"github.com/OdyseeTeam/odysee-api/pkg/logging/zapadapter"
	"github.com/Pallinder/go-randomdata"
	"github.com/gorilla/mux"
	"github.com/volatiletech/sqlboiler/queries/qm"
	"github.com/ybbus/jsonrpc/v2"

	"github.com/stretchr/testify/suite"
)

type asynqueryHandlerSuite struct {
	suite.Suite

	userHelper *e2etest.UserTestHelper
	router     *mux.Router
	launcher   *Launcher
}

func TestAsynqueryHandlerSuite(t *testing.T) {
	suite.Run(t, new(asynqueryHandlerSuite))
}

func (s *asynqueryHandlerSuite) TestPublicKey() {
	ts := httptest.NewServer(s.router)
	defer ts.Close()

	pubKey, err := keybox.PublicKeyFromURL(ts.URL + "/api/v1/asynqueries/auth/pubkey")
	s.Require().NoError(err)
	s.Require().IsType(&ecdsa.PublicKey{}, pubKey)
}

func (s *asynqueryHandlerSuite) TestCreateUpload() {
	ts := httptest.NewServer(s.router)
	defer ts.Close()

	cases := []struct {
		suffix   string
		httpTest *test.HTTPTest
	}{
		{
			suffix: "uploads",
			httpTest: &test.HTTPTest{
				Method: http.MethodPost,
				URL:    ts.URL + "/api/v1/asynqueries/uploads/",
				ReqHeader: map[string]string{
					wallet.AuthorizationHeader: s.userHelper.TokenHeader,
				},
				Code: http.StatusOK,
			},
		},
		{
			suffix: "urls",
			httpTest: &test.HTTPTest{
				Method: http.MethodPost,
				URL:    ts.URL + "/api/v1/asynqueries/urls/",
				ReqHeader: map[string]string{
					wallet.AuthorizationHeader: s.userHelper.TokenHeader,
				},
				Code: http.StatusOK,
			},
		},
	}

	for _, c := range cases {
		s.Run(c.suffix, func() {
			resp := c.httpTest.Run(s.router, s.T())

			rr := &Response{}
			s.Require().NoError(json.Unmarshal(resp.Body.Bytes(), rr))
			s.Empty(rr.Error)
			s.Require().Equal(StatusUploadTokenCreated, rr.Status)
			s.NotEmpty(rr.Payload.(UploadTokenCreatedPayload).Token)
			s.Equal(s.launcher.uploadServiceURL+c.suffix+"/", rr.Payload.(UploadTokenCreatedPayload).Location)
		})
	}
}

func (s *asynqueryHandlerSuite) TestCreate() {
	require := s.Require()
	ts := httptest.NewServer(s.router)

	uploadID := randomdata.Alphanumeric(64)
	req := jsonrpc.NewRequest(query.MethodStreamCreate, map[string]any{
		"name":                 "publish2test-dummymd",
		"title":                "Publish v2 test for dummy.md",
		"description":          "",
		"locations":            []string{},
		"bid":                  "0.01000000",
		"languages":            []string{"en"},
		"tags":                 []string{"c:disable-comments"},
		"thumbnail_url":        "https://thumbs.odycdn.com/92399dc6df41af6f7c61def97335dfa5.webp",
		"release_time":         1661882701,
		"blocking":             true,
		"preview":              false,
		"license":              "None",
		"channel_id":           "febc557fcfbe5c1813eb621f7d38a80bc4355085",
		"allow_duplicate_name": true,
		FilePathParam:          "https://uploads-v4.api.na-backend.odysee.com/v1/uploads/" + uploadID,
	})
	req.ID = randomdata.Number(1, 999999999)
	streamCreateReq, err := json.Marshal(req)
	require.NoError(err)

	createRequest := &test.HTTPTest{
		Method: http.MethodPost,
		URL:    ts.URL + "/api/v1/asynqueries/",
		ReqHeader: map[string]string{
			wallet.AuthorizationHeader: s.userHelper.TokenHeader,
		},
		ReqBody: bytes.NewReader(streamCreateReq),
		Code:    http.StatusCreated,
	}
	resp := createRequest.Run(s.router, s.T())

	var query *models.Asynquery
	e2etest.Wait(s.T(), "query settling in the database", 5*time.Second, 1000*time.Millisecond, func() error {
		mods := []qm.QueryMod{
			models.AsynqueryWhere.UploadID.EQ(uploadID),
			models.AsynqueryWhere.UserID.EQ(s.userHelper.UserID()),
		}
		query, err = models.Asynqueries(mods...).One(s.launcher.db)
		if errors.Is(err, sql.ErrNoRows) {
			return e2etest.ErrWaitContinue
		} else if err != nil {
			return err
		}
		return nil
	})

	queryReq := &jsonrpc.RPCRequest{}
	require.NoError(query.Body.Unmarshal(queryReq))
	require.Equal(req.ID, queryReq.ID)
	require.EqualValues(req.Params.(map[string]any)["name"], queryReq.Params.(map[string]any)["name"])

	rbody := resp.Body.Bytes()
	rr := &Response{}
	require.NoError(json.Unmarshal(rbody, rr))
	s.Empty(rr.Error)
	require.Equal(StatusQueryCreated, rr.Status)
	qcr := rr.Payload.(QueryCreatedPayload)
	s.Equal(query.ID, qcr.QueryID)

	s.Equal(models.AsynqueryStatusReceived, query.Status)
	s.Equal(uploadID, query.UploadID)

	(&test.HTTPTest{
		Method: http.MethodGet,
		URL:    ts.URL + "/api/v1/asynqueries/" + query.ID,
		Code:   http.StatusUnauthorized,
	}).Run(s.router, s.T())

	(&test.HTTPTest{
		Method: http.MethodGet,
		URL:    ts.URL + "/api/v1/asynqueries/" + query.ID,
		ReqHeader: map[string]string{
			wallet.AuthorizationHeader: s.userHelper.TokenHeader,
		},
		Code: http.StatusNoContent,
	}).Run(s.router, s.T())
}

func (s *asynqueryHandlerSuite) SetupSuite() {
	s.userHelper = &e2etest.UserTestHelper{}
	s.Require().NoError(s.userHelper.Setup(s.T()))
	s.router = mux.NewRouter().PathPrefix("/api/v1").Subrouter()

	kf, err := keybox.GenerateKeyfob()
	s.Require().NoError(err)

	redisHelper := testdeps.NewRedisTestHelper(s.T())
	s.launcher = NewLauncher(
		WithRequestsConnOpts(redisHelper.AsynqOpts),
		WithLogger(zapadapter.NewKV(nil)),
		WithPrivateKey(kf.PrivateKey()),
		WithDB(s.userHelper.DB),
		WithUploadServiceURL("https://uploads.odysee.com/v1/"),
	)
	s.router.Use(auth.Middleware(s.userHelper.Auther))

	err = s.launcher.InstallRoutes(s.router)
	s.Require().NoError(err)

	s.T().Cleanup(s.launcher.Shutdown)
}

type StreamCreateResponse struct {
	Height int    `json:"height"`
	Hex    string `json:"hex"`
	Inputs []struct {
		Address       string `json:"address"`
		Amount        string `json:"amount"`
		Confirmations int    `json:"confirmations"`
		Height        int    `json:"height"`
		Nout          int    `json:"nout"`
		Timestamp     int    `json:"timestamp"`
		Txid          string `json:"txid"`
		Type          string `json:"type"`
	} `json:"inputs"`
	Outputs []struct {
		Address       string `json:"address"`
		Amount        string `json:"amount"`
		ClaimID       string `json:"claim_id,omitempty"`
		ClaimOp       string `json:"claim_op,omitempty"`
		Confirmations int    `json:"confirmations"`
		Height        int    `json:"height"`
		Meta          struct {
		} `json:"meta,omitempty"`
		Name           string `json:"name,omitempty"`
		NormalizedName string `json:"normalized_name,omitempty"`
		Nout           int    `json:"nout"`
		PermanentURL   string `json:"permanent_url,omitempty"`
		Timestamp      any    `json:"timestamp"`
		Txid           string `json:"txid"`
		Type           string `json:"type"`
		Value          struct {
			Source struct {
				Hash      string `json:"hash"`
				MediaType string `json:"media_type"`
				Name      string `json:"name"`
				SdHash    string `json:"sd_hash"`
				Size      string `json:"size"`
			} `json:"source"`
			StreamType string `json:"stream_type"`
		} `json:"value,omitempty"`
		ValueType string `json:"value_type,omitempty"`
	} `json:"outputs"`
	TotalFee    string `json:"total_fee"`
	TotalInput  string `json:"total_input"`
	TotalOutput string `json:"total_output"`
	Txid        string `json:"txid"`
}
