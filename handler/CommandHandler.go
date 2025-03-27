package handler

import (
	"context"
	"fmt"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/hellodex/tradingbot/handler/callback"
	"github.com/hellodex/tradingbot/handler/commands"
	"github.com/rs/zerolog/log"
)

type (
	commandHandler = bot.HandlerFunc
	command        = string
)

const (
	start   command = "/start"
	menu    command = "/menu"
	assets  command = "/assets"
	wallets command = "/wallets"
	// invite  command = "/invite"
	// help    command = "/help"
	tradeHistory    command = "/trade_history"
	transferHistory command = "/transfer_history"
	ordersHistory   command = "/order_history"
	currentOrders   command = "/current_orders"
	aiMonitor       command = "/ai_monitor"
)

type cmd struct {
	Name command
	Desc string
}

var commandDescList = []cmd{
	{Name: start, Desc: "开始使用机器人"},
	{Name: menu, Desc: "功能菜单"},
	{Name: assets, Desc: "管理资产"},
	{Name: wallets, Desc: "管理钱包"},
	{Name: tradeHistory, Desc: "历史交易记录"},
	{Name: transferHistory, Desc: "历史转账记录"},
	{Name: ordersHistory, Desc: "历史委托记录"},
	{Name: currentOrders, Desc: "当前委托"},
	{Name: aiMonitor, Desc: "AI监控"},
}

//	var commandDesc = map[string]string{
//		start:   "开始使用机器人",
//		menu:    "功能菜单",
//		assets:  "管理资产",
//		wallets: "管理钱包",
//		// invite:  "邀请信息",
//		// help:    "帮助信息",
//		tradeHistory:    "历史交易记录",
//		transferHistory: "历史转账记录",
//		ordersHistory:   "历史委托记录",
//	}
var commandHandlerMap = map[string]commandHandler{
	start:   commands.StartHandler,
	menu:    commands.StartHandler,
	assets:  callback.AssetsHandler,
	wallets: callback.WalletMenuHandler,
	// invite:  callback.InviteHandler,
	// help:    callback.HelpHandler,
	tradeHistory:    commands.TradeHistoryHandler,
	transferHistory: commands.TransferHistoryHandler,
	ordersHistory:   commands.OpenOrdersHistoryHandler,
	currentOrders:   commands.OpenOrdersHandler,
	aiMonitor:       callback.CallbackAIMonitorMenu,
}

var _ = func() any {
	for key := range commandHandlerMap {
		text := fmt.Sprintf("%s (%s) is load", "command", key)
		log.Info().Msg(text)
	}
	return nil
}()

func CommandHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.Text == "" {
		return
	}

	if handler, exists := commandHandlerMap[update.Message.Text]; exists {
		handler(ctx, b, update)
	}
}

func SetBotCommand(ctx context.Context, bots []*bot.Bot) {
	cmds := make([]models.BotCommand, 0, len(bots))

	for _, cmd := range commandDescList {
		cmds = append(cmds, models.BotCommand{
			Command:     cmd.Name,
			Description: cmd.Desc,
		})
	}

	for _, b := range bots {
		ok, err := b.SetMyCommands(ctx, &bot.SetMyCommandsParams{
			Commands: cmds,
		})
		if err != nil {
			log.Error().Err(err).Msg("bot set commands err")
		}
		if ok {
			log.Info().Msg("bot command all set!")
		}
	}
}
