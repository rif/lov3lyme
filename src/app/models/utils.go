package models

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/dchest/uniuri"
	"labix.org/v2/mgo/bson"
)

const (
	email_user     = "no-replay@lov3ly.me"
	email_password = "*****"
	ERROR          = "error"
	INFO           = "info"
	NOTICE         = "notice"
	SUCCESS        = "success"
	PWD_LENGTH     = 8
)

func Pwdgen() string {
	return uniuri.NewLen(PWD_LENGTH)
}

func SendEmail(body []byte, to ...string) (err error) {
	err = smtp.SendMail(
		"smtp.gmail.com:587",
		smtp.PlainAuth("", email_user, email_password, "smtp.gmail.com"),
		"Lov3ly Me",
		to,
		body,
	)
	return
}

type Flash struct { // flash message
	Type    string //  "notice", "info", "success", or "error"
	Message string
}

func F(m ...string) *Flash {
	return &Flash{
		Type:    m[0],
		Message: strings.Join(m[1:], " "),
	}
}

func ImageUrl(uuid, postfix string) string {
	if postfix != "" {
		return fmt.Sprintf("/%s/%s_%s.jpg", UPLOADS, uuid, postfix)
	}
	return fmt.Sprintf("/%s/%s.jpg", UPLOADS, uuid)
}

// helper function for uuid generation
func GenUUID() string {
	uuid := make([]byte, 16)
	n, err := rand.Read(uuid)
	if n != len(uuid) || err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	uuid[8] = 0x80 // variant bits see page 5
	uuid[4] = 0x40 // version 4 Pseudo Random, see page 7

	return hex.EncodeToString(uuid)
}

func RemoveOldPasswordTokens() {
	db := db_session.Clone().DB(database)
	defer db.Session.Close()
	aDayAgo, _ := time.ParseDuration("-24h")
	db.C("passwordtokens").Remove(bson.M{"createdon": bson.M{"$lt": time.Now().Add(aDayAgo)}})
}

type WilsonSorter []bson.M

func (ws WilsonSorter) Len() int {
	return len(ws)
}
func (ws WilsonSorter) Less(i, j int) bool { // reverse sorting
	return ws[i]["wilson"].(float64) > ws[j]["wilson"].(float64)
}
func (ws WilsonSorter) Swap(i, j int) {
	ws[i], ws[j] = ws[j], ws[i]
}

func Wilson(count int, sum float64) float64 {
	if count <= 0 {
		return 0
	}
	n := float64(count)
	z := 1.96 // 95% percentile of normal distribution
	k := 4.0  // (number of items) - 1
	avg := sum / n
	lower := (avg + z*z/(2*n) - z*math.Sqrt((k*avg*(1-avg)+z*z/(4*n))/n)) / (1 + k*z*z/n)
	return 1 + 4*lower
}

func Log(msg ...string) {
	go func() {
		if _, err := sentry.CaptureMessage(msg...); err != nil {
			log.Print("could not send sentry message: ", err)
		}
	}()
}

func Logf(format string, msg ...interface{}) {
	go func() {
		if _, err := sentry.CaptureMessagef(format, msg); err != nil {
			log.Print("could not send sentry message: ", err)
		}
	}()
}
