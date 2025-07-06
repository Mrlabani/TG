package commands

import (
 "fmt"
 "strings"
 "path/filepath"

 "EverythingSuckz/fsb/config"
 "EverythingSuckz/fsb/internal/utils"

 "github.com/celestix/gotgproto/dispatcher"
 "github.com/celestix/gotgproto/dispatcher/handlers"
 "github.com/celestix/gotgproto/ext"
 "github.com/celestix/gotgproto/storage"
 "github.com/celestix/gotgproto/types"
 "github.com/gotd/td/telegram/message/styling"
 "github.com/gotd/td/tg"
)

func (m *command) LoadStream(dispatcher dispatcher.Dispatcher) {
 log := m.log.Named("start")
 defer log.Sugar().Info("Loaded")
 dispatcher.AddHandler(
  handlers.NewMessage(nil, sendLink),
 )
}

func supportedMediaFilter(m *types.Message) (bool, error) {
 if not := m.Media == nil; not {
  return false, dispatcher.EndGroups
 }
 switch m.Media.(type) {
 case *tg.MessageMediaDocument:
  return true, nil
 case *tg.MessageMediaPhoto:
  return true, nil
 case tg.MessageMediaClass:
  return false, dispatcher.EndGroups
 default:
  return false, nil
 }
}

func sendLink(ctx *ext.Context, u *ext.Update) error {
 chatId := u.EffectiveChat().GetID()
 peerChatId := ctx.PeerStorage.GetPeerById(chatId)
 if peerChatId.Type != int(storage.TypeUser) {
  return dispatcher.EndGroups
 }
 if len(config.ValueOf.AllowedUsers) != 0 && !utils.Contains(config.ValueOf.AllowedUsers, chatId) {
  ctx.Reply(u, "You are not allowed to use this bot.", nil)
  return dispatcher.EndGroups
 }
 supported, err := supportedMediaFilter(u.EffectiveMessage)
 if err != nil {
  return err
 }
 if !supported {
  ctx.Reply(u, "Sorry, this message type is unsupported.", nil)
  return dispatcher.EndGroups
 }
 update, err := utils.ForwardMessages(ctx, chatId, config.ValueOf.LogChannelID, u.EffectiveMessage.ID)
 if err != nil {
  utils.Logger.Sugar().Error(err)
  ctx.Reply(u, fmt.Sprintf("Error - %s", err.Error()), nil)
  return dispatcher.EndGroups
 }
 messageID := update.Updates[0].(*tg.UpdateMessageID).ID
 doc := update.Updates[1].(*tg.UpdateNewChannelMessage).Message.(*tg.Message).Media
 file, err := utils.FileFromMedia(doc)
 if err != nil {
  ctx.Reply(u, fmt.Sprintf("Error - %s", err.Error()), nil)
  return dispatcher.EndGroups
 }
 fullHash := utils.PackFile(
  file.FileName,
  file.FileSize,
  file.MimeType,
  file.ID,
 )
 hash := utils.GetShortHash(fullHash)
 link := fmt.Sprintf("%s/stream/%d?hash=%s", config.ValueOf.Host, messageID, hash)
 text := []styling.StyledTextOption{styling.Code(link)}

 row := tg.KeyboardButtonRow{
  Buttons: []tg.KeyboardButtonClass{
   &tg.KeyboardButtonURL{
    Text: "📥 Download",
    URL:  link + "&d=true",
   },
  },
 }

 // ✅ Replaced section: Better MIME + Extension handling for Stream
 mime := strings.ToLower(file.MimeType)
 ext := strings.ToLower(filepath.Ext(file.FileName))

 isVideo := strings.HasPrefix(mime, "video/") || 
           strings.HasSuffix(ext, ".mkv") || 
           strings.HasSuffix(ext, ".mp4") || 
           strings.HasSuffix(ext, ".webm") || 
           strings.HasSuffix(ext, ".mov") || 
           strings.HasSuffix(ext, ".avi")

 isAudio := strings.HasPrefix(mime, "audio/") || 
           strings.HasSuffix(ext, ".mp3") || 
           strings.HasSuffix(ext, ".wav") || 
           strings.HasSuffix(ext, ".flac")

 isPDF := mime == "application/pdf" || strings.HasSuffix(ext, ".pdf")

 if isVideo || isAudio || isPDF {
  row.Buttons = append(row.Buttons, &tg.KeyboardButtonURL{
   Text: "▶️ Stream",
   URL:  link,
  })
 }

 markup := &tg.ReplyInlineMarkup{
  Rows: []tg.KeyboardButtonRow{row},
 }

 if strings.Contains(link, "http://localhost") {
  _, err = ctx.Reply(u, text, &ext.ReplyOpts{
   NoWebpage:        false,
   ReplyToMessageId: u.EffectiveMessage.ID,
  })
 } else {
  _, err = ctx.Reply(u, text, &ext.ReplyOpts{
   Markup:           markup,
   NoWebpage:        false,
   ReplyToMessageId: u.EffectiveMessage.ID,
  })
 }
 if err != nil {
  utils.Logger.Sugar().Error(err)
  ctx.Reply(u, fmt.Sprintf("Error - %s", err.Error()), nil)
 }
 return dispatcher.EndGroups
}
