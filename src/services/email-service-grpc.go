package services

import (
	"context"
	"log"

	"github.com/go-playground/validator/v10"
	common_proto "github.com/oceano-dev/microservices-go-common/grpc/email/client"
	common_services "github.com/oceano-dev/microservices-go-common/services"
	trace "github.com/oceano-dev/microservices-go-common/trace/otel"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type emailServiceGrpc struct {
	common_proto.UnimplementedEmailServiceServer
	emailService common_services.EmailService
}

type passwordCode struct {
	Email string `validate:"required,email"`
	Code  string `validate:"required"`
}

type supportMessage struct {
	Message string `validate:"required"`
}

func NewEmailServerGrpc(
	emailService common_services.EmailService,
) *emailServiceGrpc {
	return &emailServiceGrpc{
		emailService: emailService,
	}
}

func (s *emailServiceGrpc) SendPasswordCode(ctx context.Context, req *common_proto.PasswordCodeReq) (*common_proto.PasswordCodeRes, error) {
	ctx, span := trace.NewSpan(ctx, "emailServiceGrpc.SendPasswordCodeReq")
	defer span.End()

	email := req.GetEmail()
	code := req.GetCode()

	model := passwordCode{
		Email: email,
		Code:  code,
	}

	validator := validator.New()
	if err := validator.StructCtx(ctx, model); err != nil {
		trace.AddSpanError(span, err)
		log.Printf("emailServiceGrpc.SendPasswordCodeReq: %v", err)
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	var err error
	go func() {
		err = s.emailService.SendPasswordCode(email, code)
	}()

	if err != nil {
		return &common_proto.PasswordCodeRes{}, err
	}

	return &common_proto.PasswordCodeRes{}, nil
}

func (s *emailServiceGrpc) SendSupportMessage(ctx context.Context, req *common_proto.SupportMessageReq) (*common_proto.SupportMessageRes, error) {
	ctx, span := trace.NewSpan(ctx, "emailServiceGrpc.SendSupportMessageReq")
	defer span.End()

	message := req.GetMessage()

	model := supportMessage{
		Message: message,
	}

	validator := validator.New()
	if err := validator.StructCtx(ctx, model); err != nil {
		trace.AddSpanError(span, err)
		log.Printf("emailServiceGrpc.SendSupportMessageReq: %v", err)
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	var err error
	go func() {
		err = s.emailService.SendSupportMessage(message)
	}()

	if err != nil {
		return &common_proto.SupportMessageRes{}, err
	}

	return &common_proto.SupportMessageRes{}, nil
}
