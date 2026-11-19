package main

import (
	"log"
	"math/rand/v2"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"

	"zzezze-bot/config"
	"zzezze-bot/features"
	"zzezze-bot/quiet"
	"zzezze-bot/session"
)

// shutdownTimeout 은 종료할 때 진행 중인 판의 버튼을 끄는 데 쓰는 최대 시간이다(systemd TimeoutStopSec 보다 짧게).
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
	dg.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsGuildMessageReactions |
		discordgo.IntentMessageContent

	// 봇 전체에 판 Registry 하나를 둔다(ADR 0003). 소음 제어는 기능별 Gate 를 따로 둔다.
	reg := session.New()

	// 메시지 트리거와 [오늘 그만] 버튼이 같은 Gate 를 봐야 하므로 Nudge 는 하나만 만든다.
	nudge := features.NewNudge(quiet.New(features.NudgeCooldown))

	router := features.NewRouter()
	router.On("nudge", nudge.HandleComponent)
	router.On("chicken", features.NewChicken(reg).HandleComponent)
	router.On("timevote", features.NewTimeVote().HandleComponent)

	// 기능 핸들러는 각자 config.Enabled 로 켜짐 여부를 확인한다.
	// 메시지 원문은 로그에 남기지 않는다(운영 서버 로그에 대화가 쌓이지 않게).
	// discordgo 는 핸들러 패닉을 잡지 않으므로 전부 감싼다(버튼은 router 가 감싼다).
	dg.AddHandler(onReady)
	dg.AddHandler(features.SafeMessage("autoreply", features.NewAutoReply(quiet.New(features.MemeCooldown)).Handle))
	dg.AddHandler(features.SafeMessage("nudge", nudge.Handle))
	dg.AddHandler(router.Handle)

	if err := dg.Open(); err != nil {
		log.Fatalf("게이트웨이 연결 실패: %v", err)
	}

	log.Println("봇이 실행 중입니다. 종료하려면 Ctrl+C.")

	// 종료 신호(SIGINT/SIGTERM)가 올 때까지 대기한다.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	// 종료 순서: 판 정리(버튼 끄기, 재시작 안내) → 게이트웨이 닫기. 2차에서 그 사이에 db.Close 가 들어간다.
	log.Println("종료합니다. 진행 중인 판을 정리합니다.")
	features.Shutdown(dg, reg, rand.IntN, shutdownTimeout)
	if err := dg.Close(); err != nil {
		log.Printf("게이트웨이 종료 실패: %v", err)
	}
}

// onReady 는 봇이 게이트웨이에 접속을 마쳤을 때 한 번 호출된다.
func onReady(_ *discordgo.Session, r *discordgo.Ready) {
	log.Printf("로그인 완료: %s (ID: %s)", r.User.Username, r.User.ID)
	log.Printf("등록된 채널 수: %d", len(config.Channels))
}
