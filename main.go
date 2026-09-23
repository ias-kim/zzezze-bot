package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"

	"zzezze-bot/config"
)

func main() {
	// .env 로드 (파일이 없으면 무시하고 실제 환경변수를 사용한다).
	_ = godotenv.Load()

	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_TOKEN 환경변수가 없습니다. .env 파일을 확인하세요.")
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalf("세션 생성 실패: %v", err)
	}

	// 메시지 내용(자동 반응 등)을 읽으려면 MessageContent 인텐트가 필요하다.
	// Developer Portal -> Bot 탭에서 MESSAGE CONTENT INTENT 도 켜져 있어야 한다.
	dg.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsGuildMessageReactions |
		discordgo.IntentMessageContent

	dg.AddHandler(onReady)
	dg.AddHandler(onMessageCreate)

	if err := dg.Open(); err != nil {
		log.Fatalf("게이트웨이 연결 실패: %v", err)
	}
	defer dg.Close()

	log.Println("봇이 실행 중입니다. 종료하려면 Ctrl+C.")

	// 종료 신호(SIGINT/SIGTERM)가 올 때까지 대기한다.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("종료합니다.")
}

// onReady 는 봇이 게이트웨이에 접속을 마쳤을 때 한 번 호출된다.
func onReady(_ *discordgo.Session, r *discordgo.Ready) {
	log.Printf("로그인 완료: %s (ID: %s)", r.User.Username, r.User.ID)
	log.Printf("등록된 채널 수: %d", len(config.Channels))
}

// onMessageCreate 는 메시지가 올라올 때마다 호출된다.
// 1단계 목표: 설정에 등록된 방의 메시지인지 인식하고, 어떤 기능이 켜졌는지 로그로 확인한다.
func onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// 봇(자기 자신 포함)이 보낸 메시지는 무시한다.
	if m.Author.Bot {
		return
	}

	// 설정(config/channels.go)에 등록된 채널만 처리한다. 나머지는 무시.
	cfg, ok := config.Channels[m.ChannelID]
	if !ok {
		return
	}

	// 어느 방에서 누가 무엇을 보냈고, 그 방에 어떤 기능이 켜졌는지 확인.
	log.Printf("[%s] %s: %q  (features=%v)",
		cfg.Name, m.Author.Username, m.Content, cfg.Features)

	// TODO(2단계~): cfg.Has("...") 로 켜진 기능만 디스패치한다.
	//   if cfg.Has("autoReply") { features.HandleAutoReply(s, m) }
	//   if cfg.Has("moderation") && isCommand(m.Content) { features.HandleModeration(s, m) }
	_ = s
}
