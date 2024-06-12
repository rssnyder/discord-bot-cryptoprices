package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/bwmarrin/discordgo"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var activity string
var token, symbol, setNickname, nicknameHeader, activityMsg, status, refresh, metrics *string
var statusCode, refreshSec int
var price, activityAmt float64
var err error
var updates prometheus.Counter

func init() {
	token = flag.String("token", getEnv("TOKEN", ""), "discord bot token")
	symbol = flag.String("symbol", getEnv("SYMBOL", ""), "crypto to watch")
	setNickname = flag.String("setNickname", getEnv("SET_NICKNAME", "true"), "to update nickname")
	nicknameHeader = flag.String("nicknameHeader", getEnv("NICKNAME_HEADER", ""), "bot nickname")
	activityMsg = flag.String("activityMsg", getEnv("ACTIVITY_MSG", ""), "bot activity")
	status = flag.String("status", getEnv("STATUS", "2"), "0: playing, 1: listening, 2: watching")
	refresh = flag.String("refresh", getEnv("REFRESH", "120"), "seconds between refresh")
	metrics = flag.String("metrics", getEnv("METRICS", ":8080"), "address for prometheus metric serving")
	flag.Parse()

	if statusCode, err = strconv.Atoi(*status); err != nil {
		log.Println(err)
		statusCode = 2
	}
	if refreshSec, err = strconv.Atoi(*refresh); err != nil {
		log.Println(err)
		refreshSec = 120
	}

	updates = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "updates",
			Help: "Number of times discord has been updated",
		},
	)
	reg := prometheus.NewRegistry()
	reg.MustRegister(updates)
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	go func() {
		log.Fatal(http.ListenAndServe(*metrics, nil))
	}()
}

func main() {
	dg, err := discordgo.New("Bot " + *token)
	if err != nil {
		log.Fatal(err)
		return
	}

	err = dg.Open()
	if err != nil {
		log.Fatal(err)
		return
	}

	guilds, err := dg.UserGuilds(100, "", "")
	if err != nil {
		log.Println(err)
		*setNickname = ""
	}
	if len(guilds) == 0 {
		*setNickname = ""
	}

	ticker := time.NewTicker(time.Duration(refreshSec) * time.Second)

	p := message.NewPrinter(language.English)

	if *nicknameHeader == "" {
		*nicknameHeader = *symbol
	}
	nickname := ""

	activities := []string{"ATH", "MCAP"}
	activityOptions := len(activities)
	activitySelection := 0

	for {
		select {
		case <-ticker.C:

			if price, err = GetCryptoPrices(*symbol, ""); err != nil {
				log.Println(err)
				continue
			}
			nickname = fmt.Sprintf("%s %s", *nicknameHeader, formatNicknameUnit(price, p))

			if *setNickname != "" {
				for _, g := range guilds {
					err = dg.GuildMemberNickname(g.ID, "@me", nickname)
					if err != nil {
						log.Println(err)
					} else {
						log.Printf("Set nickname in %s: %s\n", g.Name, nickname)
						updates.Inc()
					}
				}
			} else {
				err = setActivity(dg, nickname, statusCode)
				if err != nil {
					log.Printf("Unable to set activity: %s\n", err)
				} else {
					log.Printf("Set activity: %s\n", nickname)
					updates.Inc()
				}
				continue
			}

			if activitySelection < activityOptions {
				if activityAmt, err = GetCryptoPrices(*symbol, activities[activitySelection]); err != nil {
					log.Println(err)
					continue
				}
				activity = fmt.Sprintf("%s: %s", activities[activitySelection], formatActivityUnit(activityAmt, p))
				activitySelection++
			} else {
				if *activityMsg != "" {
					activity = *activityMsg
				}
				activitySelection = 0
			}

			setActivity(dg, activity, statusCode)
			if err != nil {
				log.Printf("Unable to set activity: %s\n", err)
			} else {
				log.Printf("Set activity: %s\n", activity)
				updates.Inc()
			}
		}
	}
}

func formatNicknameUnit(raw float64, printer *message.Printer) (units string) {
	if raw > 1 {
		units = printer.Sprintf("$%.0f", raw)
	} else {
		units = printer.Sprintf("$%f", raw)
	}

	return
}

func formatActivityUnit(raw float64, printer *message.Printer) (units string) {
	switch {
	case raw < 1:
		units = printer.Sprintf("$%.6f", raw)
	case raw < 100000:
		units = printer.Sprintf("$%.0f", raw)
	case raw < 1000000:
		units = printer.Sprintf("$%.2fk", raw/1000)
	case raw < 1000000000:
		units = printer.Sprintf("$%.2fM", raw/1000000)
	case raw < 1000000000000:
		units = printer.Sprintf("$%.2fB", raw/1000000000)
	case raw < 1000000000000000:
		units = printer.Sprintf("$%.2fT", raw/1000000000000)
	}

	return
}

func setActivity(session *discordgo.Session, text string, code int) (err error) {
	switch code {
	case 0:
		err = session.UpdateGameStatus(0, text)
	case 1:
		err = session.UpdateListeningStatus(text)
	case 2:
		err = session.UpdateWatchStatus(0, text)
	}

	return
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}

	return value
}
