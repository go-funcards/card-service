package cmd

import (
	"fmt"
	"github.com/go-funcards/card-service/internal/card"
	"github.com/go-funcards/card-service/internal/card/db"
	"github.com/go-funcards/card-service/internal/config"
	"github.com/go-funcards/card-service/proto/v1"
	"github.com/go-funcards/grpc-server"
	"github.com/go-funcards/validate"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"log"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve Card Service gRPC",
	Long:  "Serve Card Service gRPC",
	Run:   executeServeCommand,
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func executeServeCommand(cmd *cobra.Command, _ []string) {
	ctx := cmd.Context()

	cfg, err := config.GetConfig(globalFlags.ConfigFile)
	if err != nil {
		log.Fatal(err)
	}

	logger, err := cfg.Log.BuildLogger(cfg.Debug)
	if err != nil {
		panic(err)
	}

	logger.Info(fmt.Sprintf("starting: %s", use))
	logger.Info(fmt.Sprintf("version: %s", version))

	validate.Default.RegisterStructRules(cfg.Rules, []any{
		v1.CreateCardRequest_Att{},
		v1.CreateCardRequest{},
		v1.UpdateCardRequest_Att{},
		v1.UpdateCardRequest{},
		v1.UpdateManyCardsRequest{},
		v1.DeleteCardRequest{},
		v1.CardsRequest{},
	}...)

	mongoDB, err := cfg.MongoDB.GetDatabase(ctx)
	if err != nil {
		panic(err)
	}

	storage, err := db.NewStorage(ctx, mongoDB, logger)
	if err != nil {
		panic(err)
	}

	register := func(server *grpc.Server) {
		v1.RegisterCardServer(server, card.NewCardService(storage, logger))
	}

	grpcserver.Start(
		ctx,
		cfg.Server.Listen.Listener(logger),
		register,
		logger,
		grpc.ChainUnaryInterceptor(grpcserver.ValidatorUnaryServerInterceptor(validate.Default)),
	)
}
