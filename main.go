package main

import (
	asn "./asn"

	"errors"
	"fmt"
	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	geoip "github.com/oschwald/geoip2-golang"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

const geoip_db_path = "./GeoIP2-City.mmdb"

func check(e error) bool {
	if e == nil {
		return false
	}
	return true
}
func die(e error) {
	if check(e) {
		log.Fatal(e)
	}
}

func whine(e error) {
	if check(e) {
		log.Printf(e.Error())
	}
}

/** GEOIP STUFF **/
type GeoIPData struct {
	DBfile string
	DB     *geoip.Reader
}

func InitGeoIP(file string) (*GeoIPData, error) {
	geoip_data := &GeoIPData{
		DBfile: file,
	}
	db, err := geoip.Open(geoip_db_path)
	geoip_data.DB = db

	return geoip_data, err
}

func (g *GeoIPData) EndGeoIP() {
	g.DB.Close()
}

func (g *GeoIPData) GetInfo(addr string) (*geoip.City, error) {
	ip := net.ParseIP(addr)
	if ip == nil {
		return nil, errors.New("Error parsing IP address")
	}
	record, err := g.DB.City(ip)
	return record, err
}

/** SOME BOT FUNCs **/
func CreateReply(msg *tg.Message, str string) tg.MessageConfig {
	new_msg := tg.NewMessage(msg.Chat.ID, str)
	new_msg.ReplyToMessageID = msg.MessageID
	new_msg.ParseMode = "markdown"
	return new_msg
}
func Start(msg *tg.Message, b *tg.BotAPI) {
	b.Send(CreateReply(msg, "HEY THERE!"))
}
func ReplyInfo(msg *tg.Message, b *tg.BotAPI, c *geoip.City, asn_info *asn.ASNIP) {

	reply := fmt.Sprintf("*City:* %s\n*Continent:* %s\n*Country:* %s\n*Timezone:* %v\n*Coordinates:* %v, %v",
		c.City.Names["en"], c.Continent.Names["en"],
		c.Country.Names["en"], c.Location.TimeZone, c.Location.Latitude,
		c.Location.Longitude)

	if asn_info != nil {
		reply = fmt.Sprintf("%s\n*ASN:* AS%d\n*CIDR:* `%s`\n*Owner:* %s",
			reply, asn_info.Number, asn_info.CIDR.String(), asn_info.Name)
	}
	b.Send(CreateReply(msg, reply))
}
func ReplyASInfo(msg *tg.Message, b *tg.BotAPI, asn_info *asn.ASNInfo) {
	var reply string

	if asn_info == nil {
		reply = "Wrong ASN?"
	} else {
		reply = fmt.Sprintf("*ASN:* AS%d\n*Owner:* %s\n*CIDRs:*```%s```",
			asn_info.Number, asn_info.Name, asn_info.CIDRs)
	}

	if len(reply) >= 4096 { /* max telegram message */
		// Reply to long so it will be parsed... ¯\_(ツ)_/¯
		reply = fmt.Sprintf("%s%s", reply[:4090], "...```")
	}
	b.Send(CreateReply(msg, reply))
}
func Idk(msg *tg.Message, b *tg.BotAPI, t string) {
	b.Send(CreateReply(msg, t))
}

/** MAIN STUFF **/
func main() {
	log.SetOutput(os.Stderr)

	// bot stuff
	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		log.Panic("TELEGRAM_TOKEN environment var not defined!")
	}

	bot, err := tg.NewBotAPI(token)
	die(err)
	log.Printf("Bot info: %v", bot.Self)

	u := tg.NewUpdate(0)
	u.Timeout = 2

	update_chan, err := bot.GetUpdatesChan(u)

	// geoip db
	gip, err := InitGeoIP(geoip_db_path)
	die(err)
	defer gip.EndGeoIP()

	for {
		time.Sleep(1 * time.Second)
		for update := range update_chan {
			if update.Message == nil {
				continue
			}

			msg := update.Message
			pos := strings.Index(msg.Text, " ")
			// is a command?
			if msg.Text != "" && msg.Text[0] != '/' || pos == -1 {
				// ignore
				// Idk(msg, bot, "wat?!")
				continue
			}

			switch msg.Text[:pos] {
			case "/geoip":
				go func() {
					ip := msg.Text[pos+1:]

					r, e := gip.GetInfo(ip) // ignore space
					whine(e)
					if r == nil {
						Idk(msg, bot, "That's not a valid IP, dude.")
						return
					}
					// get asn info about IP
					asn_info := &asn.ASNIP{}
					if asn_info.GetIPInfo(ip) == false {
						asn_info = nil
					}

					ReplyInfo(msg, bot, r, asn_info)
				}()
			case "/geoas":
				go func() {
					as := msg.Text[pos+1:]

					if asn.CheckAS(as) {
						Idk(msg, bot, "That's not a valid ASN, dude.")
						return
					}
					// get asn info about IP
					asn_info := &asn.ASNInfo{}
					if asn_info.GetASInfo(as) == false {
						asn_info = nil
					}

					ReplyASInfo(msg, bot, asn_info)
				}()
			}
		}
	}
}
