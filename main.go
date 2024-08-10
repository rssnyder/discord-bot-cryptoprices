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
var nicknameUpdates, activityUpdates, botSymbol prometheus.Counter

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

	nicknameUpdates = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nickname_updates",
			Help: "Number of times discord nickname has been updated",
		},
	)
	activityUpdates = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "activity_updates",
			Help: "Number of times discord activity has been updated",
		},
	)
	botSymbol = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name:        "bot_symbol",
			Help:        "Exposes the symbol this bot is for",
			ConstLabels: prometheus.Labels{"symbol": *symbol},
		},
	)
	reg := prometheus.NewRegistry()
	reg.MustRegister(nicknameUpdates)
	reg.MustRegister(activityUpdates)
	reg.MustRegister(botSymbol)
	botSymbol.Inc()
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

	botUser, err := dg.User("@me")
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Printf("Running as %s", botUser.ID)

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
	activityFmt := ""
	activityOptions := len(activities)
	activitySelection := 0

	defer ticker.Stop()
	for ; true; <-ticker.C {
		if price, err = GetCryptoPrices(*symbol, ""); err != nil {
			log.Println(err)
			continue
		}
		nickname = fmt.Sprintf("%s %s", *nicknameHeader, formatPriceUnit(price, p))

		if *setNickname != "" {
			for _, g := range guilds {
				err = dg.GuildMemberNickname(g.ID, "@me", nickname)
				if err != nil {
					log.Println(err)
				} else {
					log.Printf("Set nickname in %s: %s\n", g.Name, nickname)
					nicknameUpdates.Inc()
				}
			}
		} else {
			err = setActivity(dg, nickname, statusCode)
			if err != nil {
				log.Printf("Unable to set activity: %s\n", err)
			} else {
				log.Printf("Set activity: %s\n", nickname)
				activityUpdates.Inc()
			}
			continue
		}

		if activitySelection < activityOptions {
			if activityAmt, err = GetCryptoPrices(*symbol, activities[activitySelection]); err != nil {
				log.Println(err)
				continue
			}
			switch {
			case activities[activitySelection] == "ATH":
				activityFmt = formatPriceUnit(activityAmt, p)
			case activities[activitySelection] == "MCAP":
				activityFmt = formatMcapUnit(activityAmt, p)
			}
			activity = fmt.Sprintf("%s: %s", activities[activitySelection], activityFmt)
			activitySelection++
		} else {
			if *activityMsg != "" {
				activity = *activityMsg
			}
			activitySelection = 0
		}

		err = setActivity(dg, activity, statusCode)
		if err != nil {
			log.Printf("Unable to set activity: %s\n", err)
		} else {
			log.Printf("Set activity: %s\n", activity)
			activityUpdates.Inc()
		}
	}
}

func formatPriceUnit(raw float64, printer *message.Printer) (units string) {
	switch {
	case raw > 10000:
		units = printer.Sprintf("$%.0f", raw)
	case raw > 1:
		units = printer.Sprintf("$%.2f", raw)
	case raw < 0.00001:
		units = fmt.Sprintf("$0.0₅%.0f", raw*1000000000)
	case raw < 0.000001:
		units = fmt.Sprintf("$0.0₆%.0f", raw*10000000000)
	default:
		units = printer.Sprintf("$%f", raw)
	}

	return
}

func formatMcapUnit(raw float64, printer *message.Printer) (units string) {
	switch {
	case raw < 1:
		units = printer.Sprintf("$%.6f", raw)
	case raw < 1000:
		units = printer.Sprintf("$%.2f", raw)
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
