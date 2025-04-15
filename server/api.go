package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"github.com/yiffyi/xfbbroker/data"
	"github.com/yiffyi/xfbbroker/xfb"
	"gorm.io/gorm"
)

type ApiServer struct {
	db              *data.DB
	authCallbackUrl string
}

func (s *ApiServer) probeSignPay(user *data.User) (string, error) {
	payUrl, err := xfb.RechargeOnCard("10.0", user.OpenId, user.SessionId, user.YmUserId)
	if err != nil {
		return "", err
	}

	u, _ := url.Parse(payUrl)
	tranNo := u.Query().Get("tran_no")
	_, err = xfb.SignPayCheck(tranNo)
	if err != nil {
		_, jumpUrl, err := xfb.GetSignUrl(tranNo)
		if err != nil {
			return "", err
		}

		return jumpUrl, nil
	}
	return "", nil
}

func (s *ApiServer) handleAuth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		return
	}
	q := r.URL.Query()
	u, _ := url.Parse("https://auth.xiaofubao.com/auth/user/third/getCode")
	{
		q := u.Query()
		q.Set("callBackUrl", s.authCallbackUrl)
		u.RawQuery = q.Encode()
	}

	if q.Get("ymToken") == "" || q.Get("ymUserId") == "" {
		loc, err := xfb.GetRedirectLocation(u.String()) // Get the location: compatible with WeCom
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, loc, http.StatusTemporaryRedirect)
	} else {
		sess, data, err := xfb.GetUserById(q.Get("ymToken"), q.Get("ymUserId"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			u, err := s.db.SelectUserFromYmUserId(data["id"].(string))
			if err == nil {
				err = s.db.UpdateUserSessionId(u, sess)
				// w.WriteHeader(http.StatusOK)
			} else {
				_, err = s.db.CreateUser(data["userName"].(string), data["thirdOpenid"].(string), sess, data["id"].(string))
				// w.WriteHeader(http.StatusCreated)
			}

			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusOK)
			}
		}
	}
}

func (s *ApiServer) handleSignpay(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		return
	}
	q := r.URL.Query()
	sess := q.Get("sessionId")
	if len(sess) > 0 {
		user, err := s.db.SelectUserFromSessionId(sess)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		jumpUrl, err := s.probeSignPay(user)
		if err != nil {
			http.Error(w, "signPay check failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if len(jumpUrl) > 0 {
			w.Header().Set("Location", jumpUrl)
			w.WriteHeader(http.StatusCreated)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	} else {
		http.Error(w, "no sessionId provided", http.StatusBadRequest)
	}
}

func (s *ApiServer) handleGetCards(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		return
	}
	// require sessionId
	q := r.URL.Query()
	sess := q.Get("sessionId")
	if len(sess) > 0 {
		user, err := s.db.SelectUserFromSessionId(sess)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// get card info, balance
		s, newSessionId, err := xfb.GetUserDefaultLoginInfo(user.SessionId)
		if err != nil {
			http.Error(w, "unable to get user default login info: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if newSessionId != "" {
			user.SessionId = newSessionId
		}

		balance, err := xfb.GetCardMoney(user.SessionId, user.YmUserId)
		if err != nil {
			http.Error(w, "unable to query card balance: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if balance == "- - -" {
			slog.Info(`GetCardMoney returned "- - -"`)
		}

		slog.Info("Got user card info", "Username", user.Name, "Organization", s.SchoolName, "UserType", s.UserType, "Balance", balance)
		res := map[string]any{
			"schoolName": s.SchoolName,
			"userType":   s.UserType,
			"balance":    balance,
			"userName":   s.UserName,
		}
		resArr := []map[string]any{}
		resArr = append(resArr, res)

		resBuf, err := json.MarshalIndent(resArr, "", "    ")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(resBuf)
	}
}

var codepayInstances = make(map[string]*xfb.QrPayCode)

type CodePayCreateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		QrCode string `json:"qrCode"`
	} `json:"data"`
}

func (s *ApiServer) handleCodepayCreateHelper(sessionId string, w http.ResponseWriter, r *http.Request) {
	user, err := s.db.SelectUserFromSessionId(sessionId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	code, err := xfb.GenerateQrPayCode(user.SessionId)
	if err != nil {
		// print error
		slog.Error("failed to generate qr code", "error", err)
		resBuf, err := json.MarshalIndent(map[string]any{
			"success": false,
			"message": "failed to generate qr code: server internal error",
		}, "", "    ")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(resBuf)
		return
	}

	res := CodePayCreateResponse{
		Success: true,
		Message: "success",
		Data: struct {
			QrCode string `json:"qrCode"`
		}{
			QrCode: code.QRCode,
		},
	}

	resBuf, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resBuf)

	codepayInstances[code.QRCode] = code
}

func (s *ApiServer) handleCodepayCreate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		return
	}

	sess := r.URL.Query().Get("sessionId")
	if sess == "" {
		http.Error(w, "no sessionId provided", http.StatusBadRequest)
		return
	}
	s.handleCodepayCreateHelper(sess, w, r)
}

func (s *ApiServer) handleCodepayCreatePath(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		return
	}

	vars := mux.Vars(r)
	sessionId := vars["sessionId"]
	if sessionId == "" {
		http.Error(w, "sessionId required in path", http.StatusBadRequest)
		return
	}
	s.handleCodepayCreateHelper(sessionId, w, r)
}

func (s *ApiServer) handleCodepayQueryHelper(sessionId string, w http.ResponseWriter, r *http.Request) {
	_, err := s.db.SelectUserFromSessionId(sessionId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "no code provided", http.StatusBadRequest)
		return
	}

	codepay, ok := codepayInstances[code]
	if !ok {
		http.Error(w, "codepay instance not found", http.StatusNotFound)
		return
	}

	res, err := codepay.GetResult()
	if err != nil {
		http.Error(w, "failed to query codepay: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var response map[string]any
	// check if monDealCur exists
	if _, ok := res["monDealCur"]; ok {
		// monDealCur exists, it's a completed deal
		delete(codepayInstances, code)
		response = map[string]any{
			"status":  1,
			"message": "payment completed",
			"money":   res["monDealCur"],
		}
	} else {
		// monDealCur not exists, it's an unused payment code
		// check 30s limit
		if time.Now().Unix()-codepay.Creation > 30 {
			// 30s limit exceeded, remove the codepay instance
			delete(codepayInstances, code)
			response = map[string]any{
				"status":  2,
				"message": "payment code expired",
			}
		} else {
			// 30s limit not exceeded, keep the codepay instance
			response = map[string]any{
				"status":  0,
				"message": "pending",
			}
		}
	}

	resBuf, err := json.MarshalIndent(response, "", "    ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resBuf)
}

func (s *ApiServer) handleCodepayQuery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		return
	}

	sess := r.URL.Query().Get("sessionId")
	if sess == "" {
		http.Error(w, "no sessionId provided", http.StatusBadRequest)
		return
	}
	s.handleCodepayQueryHelper(sess, w, r)
}

func (s *ApiServer) handleCodepayQueryPath(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		return
	}

	vars := mux.Vars(r)
	sessionId := vars["sessionId"]
	if sessionId == "" {
		http.Error(w, "sessionId required in path", http.StatusBadRequest)
		return
	}
	s.handleCodepayQueryHelper(sessionId, w, r)
}

func (s *ApiServer) handleRecentTransactions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		return
	}

	vars := mux.Vars(r)
	sessionId := vars["sessionId"]
	if sessionId == "" {
		http.Error(w, "sessionId required in path", http.StatusBadRequest)
		return
	}

	user, err := s.db.SelectUserFromSessionId(sessionId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, transactions, err := xfb.CardQuerynoPage(user.SessionId, user.YmUserId, time.Now())
	if err != nil {
		http.Error(w, "unable to fetch recent transactions: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Limit to at most 3 transactions
	if len(transactions) > 3 {
		transactions = transactions[len(transactions)-3:]
	}

	resBuf, err := json.MarshalIndent(transactions, "", "    ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resBuf)
}

func (s *ApiServer) handleRecentTransactionsPath(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		return
	}

	vars := mux.Vars(r)
	sessionId := vars["sessionId"]
	if sessionId == "" {
		http.Error(w, "sessionId required in path", http.StatusBadRequest)
		return
	}

	s.handleRecentTransactions(w, r)
}

func CreateApiServer(db *data.DB) *mux.Router {
	r := mux.NewRouter()
	s := &ApiServer{
		db, viper.GetString("http.auth_callback"),
	}

	// For human operations:
	r.HandleFunc("/_/xfb/auth", s.handleAuth).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/_/xfb/signpay", s.handleSignpay).Methods(http.MethodGet, http.MethodOptions)
	// r.HandleFunc("/_/config", s.handleConfig).Methods(http.MethodGet, http.MethodPut, http.MethodOptions)

	// For integrations:
	r.HandleFunc("/api/v1/cards", s.handleGetCards).Methods(http.MethodGet, http.MethodOptions)

	// Codepay endpoints
	r.HandleFunc("/api/v1/codepay/create", s.handleCodepayCreate).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/api/v1/codepay/query", s.handleCodepayQuery).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/api/v1/codepay/recentTransactions", s.handleRecentTransactions).Methods(http.MethodGet, http.MethodOptions)

	// Codepay endpoints with sessionId embedded in path
	r.HandleFunc("/api/v1/codepay/{sessionId}/create", s.handleCodepayCreatePath).Methods(http.MethodGet, http.MethodPost, http.MethodOptions)
	r.HandleFunc("/api/v1/codepay/{sessionId}/query", s.handleCodepayQueryPath).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/api/v1/codepay/{sessionId}/recentTransactions", s.handleRecentTransactionsPath).Methods(http.MethodGet, http.MethodOptions)

	r.Use(mux.CORSMethodMiddleware(r))
	return r
}
