package xfb

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

func GetUserById(token, ymId string) (sessionId string, data map[string]any, err error) {
	var r XfbResponse
	sessionId, err = Post(XfbWebApp+"/user/getUserById", "", map[string]any{
		"platform": "WECHAT_H5",
		"token":    token,
		"ymId":     ymId,
	}, &r)
	if err != nil {
		return "", nil, err
	}
	data = r.Data.(map[string]any)
	return
}

func GetUserDefaultLoginInfo(sessionId string) (data *UserDefaultLoginInfo, newSessionId string, err error) {
	var r XfbResponse
	newSessionId, err = Post(XfbWebApp+"/user/defaultLogin", sessionId, map[string]any{
		"platform": "WECHAT_H5",
	}, &r)
	if err != nil {
		slog.Error("GetUserDefaultLoginInfo", "err", err)
		return nil, "", err
	}
	data = &UserDefaultLoginInfo{}
	mData, err := json.Marshal(r.Data)
	if err != nil {
		return nil, "", err
	}
	err = json.Unmarshal(mData, data)
	return
}

func GetCardMoney(sessionId, ymId string) (string, error) {
	var r XfbResponse
	_, err := Post(XfbWebApp+"/card/getCardMoney", sessionId, map[string]any{
		"ymId": ymId,
	}, &r)
	if err != nil {
		return "", err
	}

	val := r.Data.(string)
	// what's wrong with you?
	if val == "- - -" {
		slog.Debug(`GetCardMoney: "- - -" received`, "body", r)
	}

	return val, err
}

func CardQuerynoPage(sessionId, ymId string, queryTime time.Time) (total int, rows []Trans, err error) {
	var r XfbQueryTransResponse
	_, err = Post(XfbWebApp+"/routeauth/auth/route/user/cardQuerynoPage", sessionId, map[string]any{
		"queryTime": queryTime.Format("20060102"),
		"ymId":      ymId,
	}, &r)

	if err != nil {
		// TODO err
		return
	}

	err = nil
	total = r.Total
	rows = r.Rows
	return
}

func RechargeOnCard(money, openId, sessionId, ymId string) (string, error) {
	var r XfbResponse
	_, err := Post(XfbWebApp+"/order/rechargeOnCardByParam", sessionId, map[string]any{
		"openid":         openId,
		"totalMoney":     money,
		"orderRealMoney": money,
		"rechargeType":   1,
		"subappid":       "wx8fddf03d92fd6fa9",
		"schoolCode":     "20090820",
		"platform":       "WECHAT_H5",
		"sessionId":      sessionId,
		"ymId":           ymId,
	}, &r)
	if err != nil {
		return "", err
	}
	return r.Data.(string), err
}

func SignPayCheck(tranNo string) (string, error) {
	var r XfbResponse
	_, err := Post(XfbPay+"/pay/sign/signPayCheck", "", map[string]any{
		"tranNo":  tranNo,
		"payType": "WXPAY",
	}, &r)
	return r.Message, err
}

func GetSignUrl(tranNo string) (applyId string, jumpUrl string, err error) {
	var r XfbResponse
	_, err = Post(XfbPay+"/h5/pay/sign/getSignUrl", "", map[string]any{
		"payType":     "WXPAY",
		"tranNo":      tranNo,
		"signCashier": 0,
	}, &r)
	if err != nil {
		return "", "", err
	}

	d := r.Data.(map[string]any)
	applyId = d["applyId"].(string)
	jumpUrl = d["jumpUrl"].(string)
	return
}

func QuerySignApplyById(applyId string) (int, error) {
	var r XfbResponse
	_, err := Post(XfbPay+"/h5/pay/sign/querySignApplyById", "", map[string]any{
		"applyId": applyId,
	}, &r)
	if err != nil {
		return 0, err
	}

	s := r.Data.(map[string]any)["status"].(int)
	if s == 3 {
		return s, nil
	}
	if s == 4 {
		return s, errors.New("sign application failed")
	}

	// s == 1 still applying
	return s, fmt.Errorf("unknown status: %d", s)
}

func PayChoose(tranNo string) error {
	var r XfbResponse
	_, err := Post(XfbPay+"/pay/unified/choose.shtml", "", map[string]any{
		"tranNo":    tranNo,
		"payType":   "WXPAY",
		"bussiCode": "WXSIGN",
	}, &r)
	return err
}

func DoPay(tranNo string) error {
	var r XfbResponse
	_, err := Post(XfbPay+"/pay/doPay", "", map[string]any{
		"tranNo": tranNo,
	}, &r)
	return err
}
