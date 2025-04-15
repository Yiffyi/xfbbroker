package server

import (
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"time"

	"github.com/spf13/viper"
	"github.com/yiffyi/gorad/notification"
	"github.com/yiffyi/xfbbroker/data"
	"github.com/yiffyi/xfbbroker/xfb"
)

type BackgroundLoop struct {
	db                 *data.DB
	authEndpointUrl    string
	transInterval      time.Duration
	balanceInterval    time.Duration
	batchSize          int
	sessionIdErrorPool map[string]int
}

func CreateBackgroundLoop(db *data.DB) *BackgroundLoop {
	return &BackgroundLoop{
		db:              db,
		authEndpointUrl: viper.GetString("http.auth_endpoint"),
		transInterval:   time.Duration(viper.GetInt32("loop.check_transaction_interval")) * time.Second,
		balanceInterval: time.Duration(viper.GetInt32("loop.check_balance_interval")) * time.Second,
		batchSize:       16,
	}
}

func (l *BackgroundLoop) Start() {
	go l.checkBalanceLoop()
	go l.checkTransLoop()
}

func rechargeToThreshold(curBalance float64, u *data.User) error {
	if u.AutoTopupThreshold-curBalance >= 10 {
		delta := u.AutoTopupThreshold - curBalance
		if delta > 100 {
			delta = 100.0
		}

		payUrl, err := xfb.RechargeOnCard(strconv.FormatFloat(delta, 'f', 2, 64), u.OpenId, u.SessionId, u.YmUserId)
		if err != nil {
			slog.Error("unable to recharge", "err", err)
			return err
		}
		url, _ := url.Parse(payUrl)
		tranNo := url.Query().Get("tran_no")

		_, err = xfb.SignPayCheck(tranNo)
		if err != nil {
			slog.Error("signpay check failed", "err", err)
			return err
		}
		err = xfb.PayChoose(tranNo)
		if err != nil {
			slog.Error("choose signpay failed", "err", err)
			return err
		}
		err = xfb.DoPay(tranNo)
		if err != nil {
			slog.Error("unable to pay", "err", err)
			return err
		}
		slog.Info("recharge to balance", "name", u.Name, "delta", delta, "tranNo", tranNo)
	}
	return nil
}

func formatExpense(x string) string {
	f, err := strconv.ParseFloat(x, 64)
	if err != nil {
		slog.Error("formatExpense", "err", err)
		return "数据错误"
	}

	if f < 0 {
		return fmt.Sprintf("￥%.2f", -f)
	} else {
		return fmt.Sprintf("+￥%.2f", f)
	}
}

func needNotify(feeName string) bool {
	if feeName == "金额写卡" {
		return false
	}

	return true
}

func (l *BackgroundLoop) sendNotify(u *data.User, t *xfb.Trans) error {
	cs, err := l.db.SelectNotificationChannelForUser(u.ID)
	if err != nil {
		return err
	}

	for _, c := range cs {
		if c.ChannelType == "WeComBot" && len(c.Params) > 0 {
			bot := notification.WeComBot{Key: c.Params}
			msg := map[string]any{
				"msgtype": "template_card",
				"template_card": map[string]any{
					"card_type": "text_notice",
					"source": map[string]any{
						"desc": "校园卡账单",
					},
					"main_title": map[string]any{
						"title": t.Address,
						"desc":  t.FeeName,
					},
					"emphasis_content": map[string]any{
						"title": formatExpense(t.Money),
					},
					"horizontal_content_list": []map[string]string{
						{
							"keyname": "余额",
							"value":   t.AfterMon,
						},
						{
							"keyname": "流水号",
							"value":   t.Serialno,
						},
						{
							"keyname": "交易时间",
							"value":   t.Dealtime,
						},
						{
							"keyname": "到账时间",
							"value":   t.Time,
						},
					},
					"card_action": map[string]any{
						"type": 1,
						"url":  l.authEndpointUrl,
					},
				},
			}
			return bot.SendMessage(msg)
		}
	}
	return nil
}

func (l *BackgroundLoop) sendError(u *data.User, err error) error {
	cs, err := l.db.SelectNotificationChannelForUser(u.ID)
	if err != nil {
		return err
	}

	for _, c := range cs {
		if c.ChannelType == "WeComBot" && len(c.Params) > 0 {
			bot := notification.WeComBot{Key: c.Params}
			msg := map[string]any{
				"msgtype": "template_card",
				"template_card": map[string]any{
					"card_type": "text_notice",
					"source": map[string]any{
						"desc": "校园卡账单",
					},
					"main_title": map[string]any{
						"title": "请求错误",
						"desc":  u.Name,
					},
					"sub_title_text": "自动轮询已取消，点击重新授权\n" + err.Error(),
					"horizontal_content_list": []map[string]string{
						{
							"keyname": "ymId",
							"value":   u.YmUserId,
						},
					},
					"card_action": map[string]any{
						"type": 1,
						"url":  l.authEndpointUrl,
					},
				},
			}
			return bot.SendMessage(msg)
		}
	}
	return nil
}

func (l *BackgroundLoop) breaker(u *data.User) bool {
	if l.sessionIdErrorPool[u.SessionId] > 3 {
		u.EnableAutoTopup = false
		u.EnableTransNotify = false
		err := l.db.UpdateUserLoopEnabled(u)
		if err != nil {
			slog.Error("sessionId triggered breaker in BackgroundLoop, but failed to disable user", "err", err)
		}
		delete(l.sessionIdErrorPool, u.SessionId)
		return true
	} else {
		return false
	}
}

func (l *BackgroundLoop) checkTransLoop() {
	ticker := time.NewTicker(l.transInterval)
	for {
		// select {
		// case <-ticker.C:
		batch, err := l.db.SelectUsersForTransCheck(l.batchSize)
		if err != nil {
			slog.Error("checkTransLoop: database error", "err", err)
			goto contin
		}

		for _, u := range batch {
			if l.breaker(&u) {
				goto contin
			}

			total, rows, err := xfb.CardQuerynoPage(u.SessionId, u.YmUserId, time.Now())
			if err != nil {
				slog.Error("CardQuerynoPage failed", "err", err)
				l.sessionIdErrorPool[u.SessionId]++
				if l.sessionIdErrorPool[u.SessionId] <= 3 {
					l.sendError(&u, err)
				}
				goto contin
			} else {
				updated := false
				slog.Debug("check trans", "name", u.Name, "total", total)

				for i := len(rows) - 1; i >= 0; i-- {
					v := rows[i]
					s, err := strconv.Atoi(v.Serialno)
					if err != nil {
						slog.Error("bad Serialno", "err", err, "name", u.Name, "serial", v.Serialno)
						continue
					}
					if s > u.LastTransSerial {
						slog.Info("New transaction", "detail", v)

						if needNotify(v.FeeName) {
							err = l.sendNotify(&u, &v)
							if err != nil {
								slog.Error("failed to notify", "err", err)
								break
							}
						} else {
							slog.Info("skipped", "feeName", v.FeeName)
						}

						if u.LastTransSerial < s {
							u.LastTransSerial = s
						}
						updated = true
					} else {
						continue
					}
				}

				if updated {
					l.db.UpdateUserTransSerial(&u, u.LastTransSerial)
				}
			}
		}

	contin:
		<-ticker.C
		// }
	}
}

func (l *BackgroundLoop) checkBalanceLoop() {
	ticker := time.NewTicker(l.balanceInterval)
	for {
		// select {
		// case <-ticker.C:
		batch, err := l.db.SelectUsersForBalanceCheck(l.batchSize)
		if err != nil {
			slog.Error("checkBalanceLoop: database error", "err", err)
			goto contin
		}

		for _, u := range batch {
			if l.breaker(&u) {
				goto contin
			}

			s, err := xfb.GetCardMoney(u.SessionId, u.YmUserId)
			if err != nil {
				slog.Error("unable to query card balance", "err", err, "name", u.Name)
				goto fail
			}
			if s == "- - -" {
				slog.Info(`GetCardMoney returned "- - -"`)
				continue
			}

			{ // make goto work
				balance, err := strconv.ParseFloat(s, 64)
				if err != nil {
					slog.Error("unable to parse card balance", "err", err, "name", u.Name, "rawbalance", s)
					goto fail
				}
				slog.Info("check balance", "name", u.Name, "balance", balance, "threshold", u.AutoTopupThreshold)
				// fmt.Printf("%s, current: %.2f, threshold: %.2f\n", u.Name, balance, u.Threshold)
				err = rechargeToThreshold(balance, &u)
				if err != nil {
					slog.Error("unable to recharge card balance", "err", err, "name", u.Name, "balance", balance)
					goto fail
				}
			}

			// success?
			continue

		fail:
			l.sessionIdErrorPool[u.SessionId]++
			if l.sessionIdErrorPool[u.SessionId] <= 3 {
				l.sendError(&u, err)
			}
			goto contin
		}

		// case <-stop:
		// 	return
		// }

	contin:
		<-ticker.C
	}
}
