• NewClient-
  testNewClientReturnsNil- -- Tests if NewClient returns a non-nil value.
  testNewClientReturnsIClient- -- Tests if NewClient returns an IClient interface.
• Do-
  testDoWithContextNil- -- Tests if Do function handles nil context.
  testDoWithContextNotNil- -- Tests if Do function handles non-nil context.
  testDoRequestNil- -- Tests if Do function handles nil request.
  testDoRequestNotNil- -- Tests if Do function handles non-nil request.
  testDoRequestWithBodyNil- -- Tests if Do function handles nil request body.
  testDoRequestWithBodyNotNil- -- Tests if Do function handles non-nil request body.
• MakeRequestWithRetry-
  testMakeRequestWithRetrySuccess- -- Tests if MakeRequestWithRetry function succeeds with a successful response.
  testMakeRequestWithRetryFailure- -- Tests if MakeRequestWithRetry function fails with a failed response.
  testMakeRequestWithRetryMaxRetries- -- Tests if MakeRequestWithRetry function retries maxRetries times.
  testMakeRequestWithRetryTimeout- -- Tests if MakeRequestWithRetry function handles timeout.
  testMakeRequestWithRetryInvalidMethod- -- Tests if MakeRequestWithRetry function handles invalid method.
  testMakeRequestWithRetryInvalidUrl- -- Tests if MakeRequestWithRetry function handles invalid URL.
• SetTimeout-
  testSetTimeoutValidTimeout- -- Tests if SetTimeout function sets valid timeout.
  testSetTimeoutInvalidTimeout- -- Tests if SetTimeout function handles invalid timeout.
• GetTimeout-
  testGetTimeoutValidTimeout- -- Tests if GetTimeout function returns valid timeout.
  testGetTimeoutInvalidTimeout- -- Tests if GetTimeout function handles invalid timeout.