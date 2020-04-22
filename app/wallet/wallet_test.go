package wallet

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/lbryio/lbrytv/app/sdkrouter"
	"github.com/lbryio/lbrytv/config"
	"github.com/lbryio/lbrytv/internal/lbrynet"
	"github.com/lbryio/lbrytv/internal/responses"
	"github.com/lbryio/lbrytv/internal/storage"
	"github.com/lbryio/lbrytv/internal/test"
	"github.com/lbryio/lbrytv/models"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ybbus/jsonrpc"
)

const dummyUserID = 751365

func TestMain(m *testing.M) {
	dbConfig := config.GetDatabase()
	params := storage.ConnParams{
		Connection: dbConfig.Connection,
		DBName:     dbConfig.DBName,
		Options:    dbConfig.Options,
	}
	dbConn, connCleanup := storage.CreateTestConn(params)
	dbConn.SetDefaultConnection()
	defer connCleanup()

	code := m.Run()
	os.Exit(code)
}

func setupDBTables() {
	storage.Conn.Truncate([]string{"users"})
}

func dummyAPI(sdkAddress string) (string, func()) {
	reqChan := test.ReqChan()
	ts := test.MockHTTPServer(reqChan)
	go func() {
		for {
			req := <-reqChan
			responses.AddJSONContentType(req.W)
			ts.NextResponse <- fmt.Sprintf(`{
				"success": true,
				"error": null,
				"data": {
				  "user_id": %d,
				  "has_verified_email": true
				}
			}`, dummyUserID)
		}
	}()

	return ts.URL, func() {
		ts.Close()
		UnloadWallet(sdkAddress, dummyUserID)
	}
}

func TestWalletServiceRetrieveNewUser(t *testing.T) {
	srv := test.RandServerAddress(t)
	rt := sdkrouter.New(map[string]string{"a": srv})
	setupDBTables()
	url, cleanup := dummyAPI(srv)
	defer cleanup()

	u, err := GetUserWithWallet(rt, url, "abc", "")
	require.NoError(t, err, errors.Unwrap(err))
	require.NotNil(t, u)

	count, err := models.Users(models.UserWhere.ID.EQ(u.ID)).CountG()
	require.NoError(t, err)
	assert.EqualValues(t, 1, count)
	assert.True(t, u.LbrynetServerID.IsZero()) // because the server came from a config, it should not have an id set

	// now assign the user a new server thats set in the db
	//      rand.Intn(99999),
	sdk := &models.LbrynetServer{
		Name:    "testing",
		Address: "test.test.test.test",
	}
	err = u.SetLbrynetServerG(true, sdk)
	require.NoError(t, err)
	require.NotEqual(t, 0, sdk.ID)
	require.Equal(t, u.LbrynetServerID.Int, sdk.ID)

	// now fetch it all back from the db

	u2, err := GetUserWithWallet(rt, url, "abc", "")
	require.NoError(t, err, errors.Unwrap(err))
	require.NotNil(t, u2)

	sdk2, err := u.LbrynetServer().OneG()
	require.NoError(t, err)
	require.Equal(t, sdk.ID, sdk2.ID)
	require.Equal(t, sdk.Address, sdk2.Address)
	require.Equal(t, u.LbrynetServerID.Int, sdk2.ID)
}

func TestWalletServiceRetrieveNonexistentUser(t *testing.T) {
	setupDBTables()

	ts := test.MockHTTPServer(nil)
	defer ts.Close()
	ts.NextResponse <- `{
		"success": false,
		"error": "could not authenticate user",
		"data": null
	}`

	rt := sdkrouter.New(config.GetLbrynetServers())
	u, err := GetUserWithWallet(rt, ts.URL, "non-existent-token", "")
	require.Error(t, err)
	require.Nil(t, u)
	assert.Equal(t, "cannot authenticate user with internal-apis: could not authenticate user", err.Error())
}

func TestWalletServiceRetrieveExistingUser(t *testing.T) {
	srv := test.RandServerAddress(t)
	rt := sdkrouter.New(map[string]string{"a": srv})
	setupDBTables()
	url, cleanup := dummyAPI(srv)
	defer cleanup()

	u, err := GetUserWithWallet(rt, url, "abc", "")
	require.NoError(t, err)
	require.NotNil(t, u)

	u, err = GetUserWithWallet(rt, url, "abc", "")
	require.NoError(t, err)
	assert.EqualValues(t, dummyUserID, u.ID)

	count, err := models.Users().CountG()
	require.NoError(t, err)
	assert.EqualValues(t, 1, count)
}

func TestGetUserWithWallet_ExistingUserWithSDKGetsAssignedOneOnRetrieve(t *testing.T) {
	setupDBTables()

	userID := int(rand.Int31())

	reqChan := test.ReqChan()
	ts := test.MockHTTPServer(reqChan)
	defer ts.Close()
	go func() {
		req := <-reqChan
		responses.AddJSONContentType(req.W)
		ts.NextResponse <- fmt.Sprintf(`{
			"success": true,
			"error": null,
			"data": {
			  "user_id": %d,
			  "has_verified_email": true
			}
		}`, userID)
	}()

	rt := sdkrouter.New(config.GetLbrynetServers())
	u, err := createDBUser(userID)
	require.NoError(t, err)
	require.NotNil(t, u)

	u, err = GetUserWithWallet(rt, ts.URL, "abc", "")
	require.NoError(t, err)
	assert.NotEqual(t, "", u.LbrynetServerID)
}

func TestWalletServiceRetrieveNoVerifiedEmail(t *testing.T) {
	setupDBTables()

	ts := test.MockHTTPServer(nil)
	defer ts.Close()
	ts.NextResponse <- `{
		"success": true,
		"error": null,
		"data": {
		  "user_id": 111,
		  "has_verified_email": false
		}
	}`

	rt := sdkrouter.New(config.GetLbrynetServers())
	u, err := GetUserWithWallet(rt, ts.URL, "abc", "")
	assert.NoError(t, err)
	assert.Nil(t, u)
}

func BenchmarkWalletCommands(b *testing.B) {
	setupDBTables()

	reqChan := test.ReqChan()
	ts := test.MockHTTPServer(reqChan)
	defer ts.Close()
	go func() {
		req := <-reqChan
		responses.AddJSONContentType(req.W)
		ts.NextResponse <- fmt.Sprintf(`{
			"success": true,
			"error": null,
			"data": {
			  "user_id": %v,
			  "has_verified_email": true
			}
		}`, req.R.PostFormValue("auth_token"))
	}()

	walletsNum := 60
	users := make([]*models.User, walletsNum)
	rt := sdkrouter.New(config.GetLbrynetServers())
	cl := jsonrpc.NewClient(rt.RandomServer().Address)

	logger.Disable()
	sdkrouter.DisableLogger()
	logrus.SetOutput(ioutil.Discard)

	rand.Seed(time.Now().UnixNano())

	for i := 0; i < walletsNum; i++ {
		uid := int(rand.Int31())
		u, err := GetUserWithWallet(rt, ts.URL, fmt.Sprintf("%d", uid), "")
		require.NoError(b, err, errors.Unwrap(err))
		require.NotNil(b, u)
		users[i] = u
	}

	b.SetParallelism(20)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			u := users[rand.Intn(len(users))]
			res, err := cl.Call("account_balance", map[string]string{"wallet_id": u.WalletID})
			require.NoError(b, err)
			assert.Nil(b, res.Error)
		}
	})

	b.StopTimer()
}

func TestCreate_CorrectWalletID(t *testing.T) {
	// test that calling Create() sends the correct wallet id to the server
}

func TestInitializeWallet(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	userID := rand.Int()
	addr := test.RandServerAddress(t)

	err := Create(addr, userID)
	require.NoError(t, err)

	err = UnloadWallet(addr, userID)
	require.NoError(t, err)

	err = Create(addr, userID)
	require.NoError(t, err)
}

func TestCreateWalletLoadWallet(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	userID := rand.Int()
	addr := test.RandServerAddress(t)

	err := createWallet(addr, userID)
	require.NoError(t, err)

	err = createWallet(addr, userID)
	require.NotNil(t, err)
	assert.True(t, errors.Is(err, lbrynet.ErrWalletExists))

	err = UnloadWallet(addr, userID)
	require.NoError(t, err)

	err = loadWallet(addr, userID)
	require.NoError(t, err)
}
