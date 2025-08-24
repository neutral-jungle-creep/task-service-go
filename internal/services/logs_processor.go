package services

import (
	"context"
	"os"

	"task-service/pkg/logging"
)

type LogProcessor struct {
	ctx        context.Context
	cancel     context.CancelFunc
	logChan    <-chan []byte
	outputFile *os.File
	logger     *logging.Logger
}

func NewLogProcessor(
	ctx context.Context,
	logChan <-chan []byte,
	fileName string,
	logger *logging.Logger,
) (*LogProcessor, error) {
	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)

	processor := &LogProcessor{
		ctx:        ctx,
		cancel:     cancel,
		logChan:    logChan,
		outputFile: file,
		logger:     logger,
	}

	return processor, nil
}

func (p *LogProcessor) Process() error {
	for {
		select {
		case <-p.ctx.Done():
			return nil
		case b, ok := <-p.logChan:
			if ok {
				err := p.write(b)
				if err != nil {
					return err
				}
			}
		}
	}
}

func (p *LogProcessor) Stop() {
	_ = p.outputFile.Close()
	p.cancel()
}

func (p *LogProcessor) write(b []byte) error {
	_, err := p.outputFile.Write(b)
	if err != nil {
		return err
	}
	return nil
}
