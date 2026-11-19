package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"

	"zzezze-bot/config"
	"zzezze-bot/features"
	"zzezze-bot/quiet"
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
	// 기능 핸들러는 각자 config.Enabled 로 켜짐 여부를 확인한다.
	// 메시지 원문은 로그에 남기지 않는다(운영 서버 로그에 대화가 쌓이지 않게).
	// TODO(phase1 5단계): nudge 핸들러도 여기에 등록한다.
	dg.AddHandler(features.NewAutoReply(quiet.New(features.MemeCooldown)).Handle)

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
