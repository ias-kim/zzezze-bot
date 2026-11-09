package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"

	"zzezze-bot/features"
)

// 종료할 때 진행 중인 판을 정리하는 데 쓰는 최대 시간 (docs/design/six-member.md "종료 순서").
const shutdownTimeout = 5 * time.Second

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
	// 버튼 클릭(InteractionCreate)은 인텐트 없이 온다.
	dg.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentMessageContent

	bot := features.New()
	dg.AddHandler(onReady)
	dg.AddHandler(bot.OnMessage)
	dg.AddHandler(bot.OnInteraction)

	if err := dg.Open(); err != nil {
		log.Fatalf("게이트웨이 연결 실패: %v", err)
	}

	log.Println("봇이 실행 중입니다. 종료하려면 Ctrl+C.")

	// 종료 신호(SIGINT/SIGTERM)가 올 때까지 대기한다.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	// 종료 순서: 진행 중인 판 정리(버튼 끄기 + 재시작 안내) → (2차: db.Close) → 게이트웨이 종료.
	log.Println("종료합니다.")
	bot.Shutdown(dg, shutdownTimeout)
	if err := dg.Close(); err != nil {
		log.Printf("게이트웨이 종료 실패: %v", err)
	}
}

// onReady 는 봇이 게이트웨이에 접속을 마쳤을 때 한 번 호출된다.
func onReady(_ *discordgo.Session, r *discordgo.Ready) {
	log.Printf("로그인 완료: %s (ID: %s), 서버 %d개", r.User.Username, r.User.ID, len(r.Guilds))
}
