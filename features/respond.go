package features

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

// 버튼 상호작용 응답 헬퍼. 디스코드는 3초 안에 응답을 요구하므로 핸들러는 판단 직후 이 중 하나를 부른다.
// 여러 기능(nudge, chicken, timevote)이 같이 쓰므로 기능 파일에 따로 만들지 않는다.

// noMentions 는 봇 메시지가 아무도 호출(ping)하지 않게 한다.
var noMentions = &discordgo.MessageAllowedMentions{}

// replyEphemeral 은 누른 사람에게만 보이는 메시지로 응답한다(만료·거절 안내용).
func replyEphemeral(s *discordgo.Session, i *discordgo.InteractionCreate, text string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:         text,
			Flags:           discordgo.MessageFlagsEphemeral,
			AllowedMentions: noMentions,
		},
	})
	if err != nil {
		log.Printf("ephemeral 응답 실패: %v", err)
	}
}

// updateMessage 는 버튼이 달린 원래 메시지를 고쳐서 응답한다. components 가 nil 이면 버튼을 모두 없앤다.
func updateMessage(s *discordgo.Session, i *discordgo.InteractionCreate, content string, components []discordgo.MessageComponent) {
	if components == nil {
		components = []discordgo.MessageComponent{}
	}
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:         content,
			Components:      components,
			AllowedMentions: noMentions,
		},
	})
	if err != nil {
		log.Printf("메시지 갱신 응답 실패: %v", err)
	}
}

// editMessage 는 상호작용 없이(타이머·종료 처리) 메시지를 고친다. 상호작용 토큰은 15분 뒤 만료되므로 이쪽을 쓴다.
// components 가 nil 이면 버튼을 모두 없앤다.
func editMessage(s *discordgo.Session, channelID, msgID, content string, components []discordgo.MessageComponent) error {
	if components == nil {
		components = []discordgo.MessageComponent{}
	}
	_, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		ID:              msgID,
		Channel:         channelID,
		Content:         &content,
		Components:      &components,
		AllowedMentions: noMentions,
	})
	return err
}

// userID 는 상호작용을 누른 사람의 ID 다(서버에서는 Member, DM 에서는 User).
func userID(i *discordgo.InteractionCreate) string {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User.ID
	}
	if i.User != nil {
		return i.User.ID
	}
	return ""
}

// mention 은 유저 표시 문자열이다. noMentions 와 함께 쓰면 알림 없이 이름만 보인다.
func mention(userID string) string { return "<@" + userID + ">" }
