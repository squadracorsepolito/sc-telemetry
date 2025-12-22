package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/FerroO2000/goccia"
	"github.com/FerroO2000/goccia/connector"
	"github.com/FerroO2000/goccia/egress"
	"github.com/FerroO2000/goccia/ingress"
	"github.com/FerroO2000/goccia/processor"
	"github.com/squadracorsepolito/sc-telemetry/internal"
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

	if config.Telemetry.Enabled {
		// Telemetry setup
		telemetry := pkg.NewTelemetry(config.ServiceName)
		if err := telemetry.Init(ctx, config.Telemetry); err != nil {
			log.Print("cannot initialize telemetry, use noop: ", err)
		}
		defer telemetry.Close()
	}

	connectorSize := config.ConnectorSize

	// Connectors setup
	udpToCannelloni := connector.NewRingBuffer[*ingress.UDPMessage](connectorSize)
	cannelloniToROB := connector.NewRingBuffer[*processor.CannelloniMessage](connectorSize)
	robToCAN := connector.NewRingBuffer[*processor.CannelloniMessage](connectorSize)
	canToHandler := connector.NewRingBuffer[*processor.CANMessage](connectorSize)
	canToQuestDB := connector.NewRingBuffer[*egress.QuestDBMessage](connectorSize)

	// Setup stages
	udpStage := ingress.NewUDPStage(udpToCannelloni, config.UDP.GetStageConfig())
	cannelloniStage := processor.NewCannelloniDecoderStage(udpToCannelloni, cannelloniToROB, config.Cannelloni.GetStageConfig())
	robStage := processor.NewROBStage(cannelloniToROB, robToCAN, config.ROB.GetStageConfig())
	canStage := processor.NewCANStage(robToCAN, canToHandler, config.CAN.GetStageConfig())
	handlerStage := processor.NewCustomStage(
		internal.NewCANMessageHandler(), canToHandler, canToQuestDB, config.CANMessageHandler.GetStageConfig(),
	)
	questDBStage := egress.NewQuestDBStage(canToQuestDB, config.QuestDB.GetStageConfig())

	pipeline := goccia.NewPipeline()

	pipeline.AddStage(udpStage)
	pipeline.AddStage(cannelloniStage)
	pipeline.AddStage(robStage)
	pipeline.AddStage(canStage)
	pipeline.AddStage(handlerStage)
	pipeline.AddStage(questDBStage)

	if err := pipeline.Init(ctx); err != nil {
		panic(err)
	}

	go pipeline.Run(ctx)
	defer pipeline.Close()

	<-ctx.Done()
}
