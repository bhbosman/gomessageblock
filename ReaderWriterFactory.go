package gomessageblock

import (
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type IReaderWriterFactoryService interface {
	Create() iMultiBlock
	CreateAndAddBuffer(buffer []byte) (iMultiBlock, error)
}

type ReaderWriterFactoryService struct {
}

func (r ReaderWriterFactoryService) Create() iMultiBlock {
	return NewReaderWriter()
}

func (self *ReaderWriterFactoryService) CreateAndAddBuffer(buffer []byte) (iMultiBlock, error) {
	result := NewReaderWriterSize(len(buffer))
	_, err := result.Write(buffer)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func NewReaderWriterFactoryService() *ReaderWriterFactoryService {
	return &ReaderWriterFactoryService{}
}

type ReaderWriterFactoryInParams struct {
	fx.In
	LogFactory *zap.Logger
}
type ReaderWriterFactoryOutParams struct {
	fx.Out
	ReaderWriterFactory         IReaderWriterFactoryService
	ReaderWriterFactoryInstance *ReaderWriterFactoryService
}

func ProvideReaderWriterFactoryService() fx.Option {
	return fx.Provide(
		func(params ReaderWriterFactoryInParams) ReaderWriterFactoryOutParams {
			result := NewReaderWriterFactoryService()
			return ReaderWriterFactoryOutParams{
				ReaderWriterFactory:         result,
				ReaderWriterFactoryInstance: result,
			}
		})
}
