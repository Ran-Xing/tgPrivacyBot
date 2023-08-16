package main

import (
	"crypto/tls"
	"fmt"
	"github.com/fatih/color"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"gorm.io/driver/mysql"
	"gorm.io/gorm/logger"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/deta/deta-go/deta"
	"github.com/deta/deta-go/service/base"
	. "gopkg.in/telebot.v3"
	"gorm.io/gorm"
)

// Config struct for toml config file
type (
	Mysql struct {
		UseMysql    string
		MysqlConfig string
		DB          *gorm.DB
	}
	DetaBase struct {
		UseDetaBase  string
		DetaBaseKey  string
		DetaBaseName string
		DB           *base.Base
	}
	Config struct {
		Bot           *Bot
		Token         string
		SenderID      int64 `json:"id"`
		SenderChat    Chat
		SenderType    string
		StartMessage  string
		HelpMessage   string
		GroupMessage  string
		HealthMessage string
		AdminID       string
		Mysql         Mysql
		DetaBase      DetaBase
	}
)

type MysqlApp struct {
	CreatedAt time.Time
	UserName  string `gorm:"column:user_name"`
	UserID    int64  `gorm:"column:user_id"`
	GroupID   int64  `gorm:"column:group_id"`
	GroupName string `gorm:"column:group_name"`
	Message   string `gorm:"column:message;size:5000"`
}

var (
	config      Config
	bot         *Bot
	logTemplate string
	err         error
	m           MysqlApp
	stoped      = false
)

func init() {
	// date
	log.SetPrefix(color.New(color.FgYellow).Sprintf(time.Now().In(time.FixedZone("CST", 8*3600)).Format("2006/01/02 15:04:05 ")))
	log.SetFlags(0)

	config.Token = getEnvDefault("TOKEN", "")

	if err = testToken(); err != nil {
		log.Fatal(err)
	}
	//log.Prefix("")

	if n, err := strconv.Atoi(getEnvDefault("SEND_ID", "")); err == nil {
		config.SenderID = int64(n)
	} else {
		log.Printf("SEND_TO_GROUP_ID:[%v] is not an integer.", n)
	}
	config.SenderType = getEnvDefault("SEND_TYPE", "group")
	config.SenderChat = Chat{
		ID:   config.SenderID,
		Type: "group",
	}

	if os.Getenv("NO_PROXY") != "" || os.Getenv("no_proxy") != "" {
		log.Printf("NO_PROXY")
	} else {
		if os.Getenv("HTTP_PROXY") != "" {
			log.Printf("HTTP_PROXY: %v ", os.Getenv("HTTP_PROXY"))
		}
		if os.Getenv("HTTPS_PROXY") != "" {
			log.Printf("HTTPS_PROXY: %v ", os.Getenv("HTTPS_PROXY"))
		}
		if os.Getenv("ALL_PROXY") != "" {
			log.Printf("ALL_PROXY: %v ", os.Getenv("ALL_PROXY"))
		}
		if os.Getenv("http_proxy") != "" {
			log.Printf("http_proxy: %v ", os.Getenv("http_proxy"))
		}
		if os.Getenv("https_proxy") != "" {
			log.Printf("https_proxy: %v ", os.Getenv("https_proxy"))
		}
		if os.Getenv("all_proxy") != "" {
			log.Printf("all_proxy: %v ", os.Getenv("all_proxy"))
		}
	}

	config.Mysql.UseMysql = getEnvDefault("USE_MYSQL", "no")
	if config.Mysql.UseMysql == "yes" {
		log.Printf("Mysql Enable!")
		config.Mysql.MysqlConfig = getEnvDefault("MYSQL_CONFIG", "user:passwd@tcp(ip:port)/database_name?charset=utf8mb4&parseTime=True&loc=Local")
		config.Mysql.DB, err = gorm.Open(mysql.New(mysql.Config{
			DSN:                       config.Mysql.MysqlConfig, // DSN data source name
			DefaultStringSize:         256,                      // string 类型字段的默认长度
			DisableDatetimePrecision:  true,                     // 禁用 datetime 精度，MySQL 5.6 之前的数据库不支持
			DontSupportRenameIndex:    true,                     // 重命名索引时采用删除并新建的方式，MySQL 5.7 之前的数据库和 MariaDB 不支持重命名索引
			DontSupportRenameColumn:   true,                     // 用 `change` 重命名列，MySQL 8 之前的数据库和 MariaDB 不支持重命名列
			SkipInitializeWithVersion: false,                    // 根据当前 MySQL 版本自动配置
		}), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err = config.Mysql.DB.AutoMigrate(&MysqlApp{}); err != nil {
			log.Fatalf("自动创建表失败, %s", err)
			return
		}
	}

	config.DetaBase.UseDetaBase = getEnvDefault("USE_DETA_BASE", "no")
	if config.DetaBase.UseDetaBase == "yes" {
		log.Printf("DetaBase Enable!")
		config.DetaBase.DetaBaseKey = getEnvDefault("DETA_BASE_KEY", "")
		config.DetaBase.DetaBaseName = getEnvDefault("DETA_BASE_NAME", "")
		if config.DetaBase.DetaBaseKey == "" || config.DetaBase.DetaBaseName == "" {
			log.Fatalf("DetaBase Key or Name is empty!")
			return
		}
		// initialize with project key
		d, err := deta.New(deta.WithProjectKey("project_key"))
		if err != nil {
			log.Fatalf("failed to init new Deta instance: %v", err)
			return
		}

		// initialize with base name
		config.DetaBase.DB, err = base.New(d, "MysqlApp")
		if err != nil {
			log.Fatalf("failed to init new Base instance: %v", err)
			return
		}
	}

	config.StartMessage = getEnvDefault("START_MESSAGE", "welcome!")
	config.HelpMessage = getEnvDefault("HELP_MESSAGE", "help")
	config.HealthMessage = getEnvDefault("HEALTH_MESSAGE", "health")
	config.GroupMessage = getEnvDefault("GROUP_MESSAGE", "group")
	config.AdminID = getEnvDefault("ADMIN_ID", "")
}

func main() {
	httpClient := privacyDns()

	bot, err = NewBot(Settings{
		Token:  config.Token,
		Poller: &LongPoller{Timeout: 10 * time.Second},
		Client: httpClient,
	})
	if err != nil {
		log.Fatalf("Cannot start bot. Error: %v\n", err)
	}

	bot.Handle("/start", func(c Context) error {
		tgLog(c, "group")
		if !c.Message().Private() || stoped {
			log.Printf("skip group message")
			return nil
		}
		if _, err := bot.Send(c.Chat(), config.StartMessage+"\n"+config.HelpMessage); err != nil {
			log.Println(err)
			return err
		}
		return err
	})
	bot.Handle("/health", func(c Context) error {
		tgLog(c, "health")
		if !c.Message().Private() || stoped {
			log.Printf("skip group message")
			return nil
		}
		if _, err := bot.Send(c.Chat(), config.HealthMessage); err != nil {
			log.Println(err)
			return err
		}
		return err
	})
	bot.Handle("/help", func(c Context) error {
		tgLog(c, "help")
		if !c.Message().Private() || stoped {
			log.Printf("skip group message")
			return nil
		}
		if _, err := bot.Send(c.Chat(), config.HelpMessage); err != nil {
			log.Println(err)
			return err
		}
		return err
	})
	bot.Handle("/group", func(c Context) error {
		tgLog(c, "group")
		if !c.Message().Private() || stoped {
			log.Printf("skip group message")
			return nil
		}
		if _, err := bot.Send(c.Chat(), config.GroupMessage); err != nil {
			log.Println(err)
			return err
		}
		return err
	})
	bot.Handle("/enable", func(c Context) error {
		tgLog(c, "enable")
		if !c.Message().Private() || stoped {
			log.Printf("skip group message")
			return nil
		}
		if config.AdminID != "" && strconv.FormatInt(c.Sender().ID, 10) != config.AdminID {
			if _, err := bot.Send(c.Chat(), "你没有权限执行此操作!"); err != nil {
				log.Println(err)
			}
			return err
		}

		if _, err := bot.Send(c.Chat(), "Bot 服务已启动!"); err != nil {
			log.Println(err)
		}
		//os.Exit(1)
		stoped = false
		return err
	})
	bot.Handle("/disable", func(c Context) error {
		tgLog(c, "disable")
		if !c.Message().Private() || stoped {
			log.Printf("skip group message")
			return nil
		}
		if config.AdminID != "" && strconv.FormatInt(c.Sender().ID, 10) != config.AdminID {
			if _, err := bot.Send(c.Chat(), "你没有权限执行此操作!"); err != nil {
				log.Println(err)
			}
			return err
		}

		if _, err := bot.Send(c.Chat(), "Bot 服务已停止!"); err != nil {
			log.Println(err)
		}
		//os.Exit(1)
		stoped = true
		return err
	})
	bot.Handle(OnText, forwardMessage)
	bot.Handle(OnPhoto, forwardMessage)
	bot.Handle(OnDocument, forwardMessage)
	bot.Handle(OnSticker, forwardMessage)
	bot.Handle(OnAnimation, forwardMessage)
	bot.Handle(OnVenue, forwardMessage)
	bot.Handle(OnMedia, func(c Context) error {
		tgLog(c, "forward media")
		if !c.Message().Private() || stoped {
			log.Printf("skip group message")
			return nil
		}
		if _, err = bot.Send(&config.SenderChat, c.Text()+"\n\n附件无法发送"); err != nil {
			log.Println(err)
		}
		return err
	})
	log.Println("Bot started!")
	bot.Start()
	//go func() {
	//	bot.Start()
	//}()
	//
	//signalChan := make(chan os.Signal, 1)
	//signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	//<-signalChan
	//log.Println("Shutdown signal received, exiting...")
}

// 在接收到要转发的消息时调用此回调函数
func forwardMessage(c Context) error {
	tgLog(c, "forward")
	if !c.Message().Private() || stoped {
		log.Printf("skip group message")
		return nil
	}
	if ok, _ := regexp.MatchString("^(\\d+)$", c.Text()); ok {
		if _, err := bot.Send(c.Chat(), "Bot 不提供查询功能"); err != nil {
			log.Println(err)
		}
		return err
	}
	if msg, err := bot.Send(&config.SenderChat, c.Text()); err != nil {
		log.Println("forwardMessage error", err)
		return err
	} else {
		if _, err := bot.Send(c.Chat(), fmt.Sprintf("[Forward success!](t.me/%s/%d)", msg.Chat.Username, msg.ID), &SendOptions{ParseMode: "markdown"}); err != nil {
			log.Println("replyMessage error", err)
			return err
		}
	}
	return err
}

func getEnvDefault(key, defVal string) string {
	val, ex := os.LookupEnv(key)
	if !ex {
		return defVal
	}
	return val
}

func testToken() (err error) {
	if config.Token == "" {
		err = errors.Errorf("Env variable %v isn't set!", config.Token)
		return err
	}
	match, err := regexp.MatchString(`^[0-9]+:.*$`, config.Token)
	if err != nil {
		return err
	}
	if !match {
		err = errors.Errorf("Telegram Bot Token [%v] is incorrect. Token doesn't comply with regexp: `^[0-9]+:.*$`. Please, provide a correct Telegram Bot Token through env variable TGTOKEN", config.Token)
		return err
	}
	return err
}

func tgLog(c Context, msg string) {
	logTemplate = fmt.Sprintf("user:{%v %v %v} in chat:[%v]", c.Sender().FirstName, c.Sender().LastName, c.Chat().ID, c.Text())
	m = MysqlApp{
		UserName: c.Sender().FirstName + " " + c.Sender().LastName,
		UserID:   c.Chat().ID,
		Message:  c.Text(),
	}

	if c.Chat().Type == "group" {
		logTemplate = fmt.Sprintf("user:{%v %v %v} group:{%v %v} in chat:[%v]", c.Sender().FirstName, c.Sender().LastName, c.Sender().ID, c.Chat().Title, c.Chat().ID, c.Text())
		m.GroupName = c.Chat().Title
		m.GroupID = c.Chat().ID
	}
	log.Printf("%v request from %v", msg, logTemplate)

	if fmt.Sprintf("%v", m.UserID) == config.AdminID {
		log.Printf("Admin skip")
		return
	}

	if config.DetaBase.UseDetaBase == "yes" {
		if _, err = config.DetaBase.DB.Insert(&m); err != nil {
			log.Fatalf("写入数据失败: %v\n", err)
			return
		}
	}
	if config.Mysql.UseMysql == "yes" {
		if err = config.Mysql.DB.Create(&m).Error; err != nil {
			log.Fatalf("写入数据失败, %s", err)
			return
		}
	}
}

func privacyDns() (client *http.Client) {
	// 设置制定DNS 保护隐私
	dialer := &net.Dialer{
		Resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: time.Duration(5000) * time.Millisecond,
				}
				return d.DialContext(ctx, "udp", "8.8.8.8:53")
			},
		},
	}
	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialer.DialContext(ctx, network, addr)
	}
	client = &http.Client{
		Timeout: 50 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			Proxy:           http.ProxyFromEnvironment,
			DialContext:     dialContext,
		},
	}
	return client
}
