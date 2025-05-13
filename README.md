# mongo-spec-gpt

Run `make` to build `mongo-spec-gpt` and put it in the GOPATH `bin` directory. This requires Go 1.23 or higher.

Required environment variables:

- `MONGODB_URI`: Atlas cluster 
- `OPENAI_API_KEY`: OpenAI API key
- `OPENAI_BASE_URI`: URI to openAI API endpoint 

LangchainGo's defaults: https://github.com/tmc/langchaingo/blob/42487bafecc6b843e1440bdf57701e2296f766ef/llms/openai/openaillm_option.go#L9
