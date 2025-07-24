package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/squadracorsepolito/acmetel"
	"github.com/squadracorsepolito/acmetel/can"
	"github.com/squadracorsepolito/acmetel/cannelloni"
	"github.com/squadracorsepolito/acmetel/connector"
	"github.com/squadracorsepolito/acmetel/questdb"
	"github.com/squadracorsepolito/acmetel/udp"
	"github.com/squadracorsepolito/sc-telemetry/pkg"
)

func main() {
	ctx, cancelCtx := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer cancelCtx()

	// Load the config
	config, err := pkg.LoadConfig()
	if err != nil {
		panic(err)
	}

	// Telemetry setup
	telemetry := pkg.NewTelemetry(config.ServiceName)
	if err := telemetry.Init(ctx); err != nil {
		panic(err)
	}
	defer telemetry.Close()

	// Connectors setup
	udpToCannelloni := connector.NewRingBuffer[*udp.Message](uint32(config.Connectors.UDPSize))
	cannelloniToCAN := connector.NewRingBuffer[*cannelloni.Message](uint32(config.Connectors.CannelloniSize))
	canToQuestDB := connector.NewRingBuffer[*can.Message](uint32(config.Connectors.CANSize))

	// First stage: UDP ingress
	udpCfg := config.Stages.UDP
	udpIngress := udp.NewStage(udpToCannelloni, udpCfg)

	// Second stage: cannelloni handler
	cannelloniCfg := cannelloni.NewDefaultConfig()
	cannelloniHandler := cannelloni.NewStage(udpToCannelloni, cannelloniToCAN, cannelloniCfg)

	// Third stage: CAN handler
	canMessages, err := pkg.GetMessagesFromDBC(config.DBCFilePath)
	if err != nil {
		panic(err)
	}

	canCfg := config.Stages.CAN
	canCfg.Messages = canMessages
	canHandler := can.NewStage(cannelloniToCAN, canToQuestDB, canCfg)

	// Fourth stage: QuestDB egress
	questDBCfg := config.Stages.QuestDB
	questDBEgress := questdb.NewStage(pkg.NewQuestDBHandler(), canToQuestDB, questDBCfg)

	// Pipeline setup
	pipeline := acmetel.NewPipeline()

	pipeline.AddStage(udpIngress)
	pipeline.AddStage(cannelloniHandler)
	pipeline.AddStage(canHandler)
	pipeline.AddStage(questDBEgress)

	if err := pipeline.Init(ctx); err != nil {
		panic(err)
	}

	go pipeline.Run(ctx)
	defer pipeline.Stop()

	<-ctx.Done()
}
