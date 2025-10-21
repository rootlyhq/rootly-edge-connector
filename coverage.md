# Test Coverage Report

**Generated:** 2025-11-05
**Total Coverage:** 77.0%
**Total Tests:** 384

## Coverage by Package

| Package | Coverage |
|---------|----------|
| client.go:145: | 100.0% |
| client.go:162: | 68.3% |
| client.go:262: | 66.7% |
| client.go:26: | 100.0% |
| client.go:271: | 61.9% |
| client.go:382: | 59.5% |
| client.go:43: | 75.7% |
| config.go:189: | 100.0% |
| config.go:211: | 100.0% |
| config.go:230: | 100.0% |
| config.go:265: | 100.0% |
| config.go:317: | 100.0% |
| config.go:324: | 100.0% |
| config.go:334: | 100.0% |
| config.go:348: | 100.0% |
| converter.go:50: | 100.0% |
| converter.go:55: | 100.0% |
| converter.go:9: | 100.0% |
| executor.go:142: | 100.0% |
| executor.go:152: | 100.0% |
| executor.go:216: | 100.0% |
| executor.go:259: | 100.0% |
| executor.go:278: | 94.1% |
| executor.go:322: | 100.0% |
| executor.go:34: | 100.0% |
| executor.go:44: | 92.7% |
| http.go:268: | 100.0% |
| http.go:285: | 100.0% |
| http.go:293: | 100.0% |
| http.go:39: | 100.0% |
| http.go:48: | 95.3% |
| loader.go:167: | 53.8% |
| loader.go:19: | 95.0% |
| loader.go:64: | 91.7% |
| loader.go:92: | 95.0% |
| main.go:246: | 0.0% |
| main.go:29: | 0.0% |
| main.go:344: | 0.0% |
| manager.go:129: | 87.5% |
| manager.go:148: | 72.4% |
| manager.go:202: | 82.4% |
| manager.go:237: | 100.0% |
| manager.go:251: | 100.0% |
| manager.go:262: | 50.0% |
| manager.go:284: | 100.0% |
| manager.go:291: | 77.8% |
| manager.go:310: | 100.0% |
| manager.go:43: | 100.0% |
| manager.go:51: | 89.3% |
| metrics.go:189: | 100.0% |
| metrics.go:202: | 100.0% |
| metrics.go:215: | 100.0% |
| metrics.go:221: | 100.0% |
| metrics.go:230: | 100.0% |
| metrics.go:239: | 100.0% |
| metrics.go:54: | 100.0% |
| poller.go:128: | 73.3% |
| poller.go:30: | 100.0% |
| poller.go:40: | 100.0% |
| poller.go:70: | 79.2% |
| pool.go:119: | 100.0% |
| pool.go:127: | 100.0% |
| pool.go:132: | 100.0% |
| pool.go:31: | 100.0% |
| pool.go:50: | 83.3% |
| pool.go:70: | 80.0% |
| pool.go:92: | 100.0% |
| reporter.go:33: | 100.0% |
| reporter.go:40: | 100.0% |
| script.go:238: | 83.3% |
| script.go:264: | 91.7% |
| script.go:33: | 100.0% |
| script.go:41: | 100.0% |
| script.go:46: | 97.3% |
| validator.go:118: | 100.0% |
| validator.go:213: | 87.5% |
| validator.go:21: | 100.0% |
| validator.go:253: | 95.7% |
| validator.go:303: | 87.5% |
| validator.go:323: | 100.0% |
| validator.go:335: | 100.0% |
| validator.go:95: | 90.0% |

## Detailed Function Coverage

```
github.com/rootly/edge-connector/cmd/rec/main.go:29:			main				0.0%
github.com/rootly/edge-connector/cmd/rec/main.go:246:			validateConfig			0.0%
github.com/rootly/edge-connector/cmd/rec/main.go:344:			initLogger			0.0%
github.com/rootly/edge-connector/internal/api/client.go:26:		NewClient			100.0%
github.com/rootly/edge-connector/internal/api/client.go:43:		FetchEvents			75.7%
github.com/rootly/edge-connector/internal/api/client.go:145:		logRateLimitHeaders		100.0%
github.com/rootly/edge-connector/internal/api/client.go:162:		RegisterActions			68.3%
github.com/rootly/edge-connector/internal/api/client.go:262:		redactToken			66.7%
github.com/rootly/edge-connector/internal/api/client.go:271:		MarkDeliveryAsRunning		61.9%
github.com/rootly/edge-connector/internal/api/client.go:382:		ReportExecution			59.5%
github.com/rootly/edge-connector/internal/api/converter.go:9:		ConvertActionsToRegistrations	100.0%
github.com/rootly/edge-connector/internal/api/converter.go:50:		generateAutomaticDescription	100.0%
github.com/rootly/edge-connector/internal/api/converter.go:55:		convertParameterDefinitions	100.0%
github.com/rootly/edge-connector/internal/config/config.go:189:		GetEventTypes			100.0%
github.com/rootly/edge-connector/internal/config/config.go:211:		ConvertToActions		100.0%
github.com/rootly/edge-connector/internal/config/config.go:230:		onActionToAction		100.0%
github.com/rootly/edge-connector/internal/config/config.go:265:		callableActionToAction		100.0%
github.com/rootly/edge-connector/internal/config/config.go:317:		getOrDefault			100.0%
github.com/rootly/edge-connector/internal/config/config.go:324:		getTimeoutOrDefault		100.0%
github.com/rootly/edge-connector/internal/config/config.go:334:		mergeEnv			100.0%
github.com/rootly/edge-connector/internal/config/config.go:348:		autoGenerateParameters		100.0%
github.com/rootly/edge-connector/internal/config/loader.go:19:		Load				95.0%
github.com/rootly/edge-connector/internal/config/loader.go:64:		LoadActions			91.7%
github.com/rootly/edge-connector/internal/config/loader.go:92:		applyDefaults			95.0%
github.com/rootly/edge-connector/internal/config/loader.go:167:		applyActionDefaults		53.8%
github.com/rootly/edge-connector/internal/config/validator.go:21:	Validate			100.0%
github.com/rootly/edge-connector/internal/config/validator.go:95:	ValidateActions			90.0%
github.com/rootly/edge-connector/internal/config/validator.go:118:	validateAction			100.0%
github.com/rootly/edge-connector/internal/config/validator.go:213:	validateParameterDefinitions	87.5%
github.com/rootly/edge-connector/internal/config/validator.go:253:	validateParameterDefaults	95.7%
github.com/rootly/edge-connector/internal/config/validator.go:303:	formatParameterValidationErrors	87.5%
github.com/rootly/edge-connector/internal/config/validator.go:323:	contains			100.0%
github.com/rootly/edge-connector/internal/config/validator.go:335:	NormalizeActionName		100.0%
github.com/rootly/edge-connector/internal/executor/executor.go:34:	New				100.0%
github.com/rootly/edge-connector/internal/executor/executor.go:44:	Execute				92.7%
github.com/rootly/edge-connector/internal/executor/executor.go:142:	findMatchingAction		100.0%
github.com/rootly/edge-connector/internal/executor/executor.go:152:	matchesAction			100.0%
github.com/rootly/edge-connector/internal/executor/executor.go:216:	prepareParameters		100.0%
github.com/rootly/edge-connector/internal/executor/executor.go:259:	substituteTemplate		100.0%
github.com/rootly/edge-connector/internal/executor/executor.go:278:	prepareTemplateContext		94.1%
github.com/rootly/edge-connector/internal/executor/executor.go:322:	getFieldValue			100.0%
github.com/rootly/edge-connector/internal/executor/http.go:39:		NewHTTPExecutor			100.0%
github.com/rootly/edge-connector/internal/executor/http.go:48:		Execute				95.3%
github.com/rootly/edge-connector/internal/executor/http.go:268:		renderTemplate			100.0%
github.com/rootly/edge-connector/internal/executor/http.go:285:		truncateString			100.0%
github.com/rootly/edge-connector/internal/executor/http.go:293:		prepareTemplateContext		100.0%
github.com/rootly/edge-connector/internal/executor/script.go:33:	NewScriptRunner			100.0%
github.com/rootly/edge-connector/internal/executor/script.go:41:	SetGitManager			100.0%
github.com/rootly/edge-connector/internal/executor/script.go:46:	Run				97.3%
github.com/rootly/edge-connector/internal/executor/script.go:238:	isAllowedPath			83.3%
github.com/rootly/edge-connector/internal/executor/script.go:264:	detectInterpreter		91.7%
github.com/rootly/edge-connector/internal/metrics/metrics.go:54:	InitMetrics			100.0%
github.com/rootly/edge-connector/internal/metrics/metrics.go:189:	NewServer			100.0%
github.com/rootly/edge-connector/internal/metrics/metrics.go:202:	Start				100.0%
github.com/rootly/edge-connector/internal/metrics/metrics.go:215:	Shutdown			100.0%
github.com/rootly/edge-connector/internal/metrics/metrics.go:221:	RecordActionExecution		100.0%
github.com/rootly/edge-connector/internal/metrics/metrics.go:230:	RecordHTTPRequest		100.0%
github.com/rootly/edge-connector/internal/metrics/metrics.go:239:	RecordGitPull			100.0%
github.com/rootly/edge-connector/internal/poller/poller.go:30:		New				100.0%
github.com/rootly/edge-connector/internal/poller/poller.go:40:		Start				100.0%
github.com/rootly/edge-connector/internal/poller/poller.go:70:		poll				79.2%
github.com/rootly/edge-connector/internal/poller/poller.go:128:		handleError			73.3%
github.com/rootly/edge-connector/internal/reporter/reporter.go:33:	New				100.0%
github.com/rootly/edge-connector/internal/reporter/reporter.go:40:	Report				100.0%
github.com/rootly/edge-connector/internal/worker/pool.go:31:		NewPool				100.0%
github.com/rootly/edge-connector/internal/worker/pool.go:50:		Start				83.3%
github.com/rootly/edge-connector/internal/worker/pool.go:70:		Submit				80.0%
github.com/rootly/edge-connector/internal/worker/pool.go:92:		worker				100.0%
github.com/rootly/edge-connector/internal/worker/pool.go:119:		Shutdown			100.0%
github.com/rootly/edge-connector/internal/worker/pool.go:127:		QueueSize			100.0%
github.com/rootly/edge-connector/internal/worker/pool.go:132:		QueueCapacity			100.0%
github.com/rootly/edge-connector/pkg/git/manager.go:43:			NewManager			100.0%
github.com/rootly/edge-connector/pkg/git/manager.go:51:			Download			89.3%
github.com/rootly/edge-connector/pkg/git/manager.go:129:		PullAll				87.5%
github.com/rootly/edge-connector/pkg/git/manager.go:148:		Pull				72.4%
github.com/rootly/edge-connector/pkg/git/manager.go:202:		GetScriptPath			82.4%
github.com/rootly/edge-connector/pkg/git/manager.go:237:		RLock				100.0%
github.com/rootly/edge-connector/pkg/git/manager.go:251:		RUnlock				100.0%
github.com/rootly/edge-connector/pkg/git/manager.go:262:		getAuth				50.0%
github.com/rootly/edge-connector/pkg/git/manager.go:284:		sanitizeRepoName		100.0%
github.com/rootly/edge-connector/pkg/git/manager.go:291:		StartPeriodicPull		77.8%
github.com/rootly/edge-connector/pkg/git/manager.go:310:		GetRepository			100.0%
total:									(statements)			77.0%
```
