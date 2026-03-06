# 🎬 Demo Report — 2026-03-06 17:18

## Service Health
```
  ❌ :14250 → UNREACHABLE
  ❌ :14268 → UNREACHABLE
  ❌ :15432 → UNREACHABLE
  ❌ :16379 → UNREACHABLE
  ✅ :16686 → <!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <!-- prevent caching of this HTML by any server, Go or otherwise -->
    <meta http-equiv="cache-control" content="max-age=0" />
    <meta http-equiv="cache-control" content="no-cache" />
    <meta http-equiv="expires" content="0" />
    <meta http-equiv="expires" content="Tue, 01 Jan 1980 1:00:00 GMT" />
    <meta http-equiv="pragma" content="no-cache" />

    <!-- NOTE: The document MUST have a <base> element. package.json#homepage is set to "." as part of resolving https://github.com/jaegertracing/jaeger-ui/issues/42 and therefore static assets are linked via relative URLs. This will break on many document URLs, e.g. /trace/abc, unless a valid base URL is provided. The base href defaults to "/" but the query-service can inject an override. -->
    <base href="/" data-inject-target="BASE_URL" />
    <link rel="shortcut icon" href="./static/favicon-BxcVf0am.ico">
    <title>Jaeger UI</title>
    <script>
      // Jaeger UI config data is embedded by the query-service via search-replace.
      // Please see ./README.md#configuration for details.

      // TODO the JSON/JS bifurcation below could be avoided by using a single template function like:
      // function getJaegerUiConfig() { return null; }

      // Important! Do not alter the following line; query-service looks for that exact pattern.
      // JAEGER_CONFIG_JS

      function getJaegerUiConfig() {
        if(typeof window.UIConfig === 'function') {
          return UIConfig();
        }
        const DEFAULT_CONFIG = null;
        // Important! Do not alter the following line; query-service looks for that exact pattern.
        const JAEGER_CONFIG = DEFAULT_CONFIG;
        return JAEGER_CONFIG;
      }
      // Jaeger storage compabilities data is embedded by the query-service via search-replace.
      function getJaegerStorageCapabilities() {
        const DEFAULT_STORAGE_CAPABILITIES = { "archiveStorage": false };
        const JAEGER_STORAGE_CAPABILITIES = {"archiveStorage":true};
        return JAEGER_STORAGE_CAPABILITIES;
      }
      // Jaeger version data is embedded by the query-service via search/replace.
      function getJaegerVersion() {
        const DEFAULT_VERSION = {'gitCommit':'', 'gitVersion':'', 'buildDate':''};
        // Important! Do not alter the following line; query-service looks for that exact pattern.
        const JAEGER_VERSION = {"gitCommit":"36f2a31de3147231ca0adcd96a0a13e6ef55ea71","gitVersion":"v1.58.1","buildDate":"2024-06-22T20:40:52Z"};
        return JAEGER_VERSION;
      }

      // Workaround some legacy NPM dependencies that assume this is always defined.
      window.global = {};
    </script>
    <script type="module" crossorigin src="./static/index-CDLMgXBK.js"></script>
    <link rel="stylesheet" crossorigin href="./static/index-bzTJ6oK_.css">
    <script type="module">import.meta.url;import("_").catch(()=>1);(async function*(){})().next();if(location.protocol!="file:"){window.__vite_is_modern_browser=true}</script>
    <script type="module">!function(){if(window.__vite_is_modern_browser)return;console.warn("vite: loading legacy chunks, syntax error above and the same error below should be ignored");var e=document.getElementById("vite-legacy-polyfill"),n=document.createElement("script");n.src=e.src,n.onload=function(){System.import(document.getElementById('vite-legacy-entry').getAttribute('data-src'))},document.body.appendChild(n)}();</script>
  </head>
  <body>
    <div id="jaeger-ui-root"></div>
    <!--
      This file is the main entry point for the Jaeger UI application.
      See https://vitejs.dev/guide/#index-html-and-project-root for more information
      on how asset references are managed by the build system.
    -->
    <script nomodule>!function(){var e=document,t=e.createElement("script");if(!("noModule"in t)&&"onbeforeload"in t){var n=!1;e.addEventListener("beforeload",(function(e){if(e.target===t)n=!0;else if(!e.target.hasAttribute("nomodule")||!n)return;e.preventDefault()}),!0),t.type="module",t.src=".",e.head.appendChild(t),t.remove()}}();</script>
    <script nomodule crossorigin id="vite-legacy-polyfill" src="./static/polyfills-legacy-D919C7DZ.js"></script>
    <script nomodule crossorigin id="vite-legacy-entry" data-src="./static/index-legacy-CpL7BSIU.js">System.import(document.getElementById('vite-legacy-entry').getAttribute('data-src'))</script>
  </body>
</html>
  ✅ :18001 → {"status":"healthy","service":"auth-service"}
  ✅ :18005 → {"status":"healthy","service":"notification-service"}
  ✅ :3000 → <!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/svg+xml" href="/assets/favicon-DqYfaDoO.svg" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>OpenPrint Cloud</title>
    <script type="module" crossorigin src="/assets/js/index-COQYgrPN.js"></script>
    <link rel="modulepreload" crossorigin href="/assets/js/vendor-react-CZqGU4pI.js">
    <link rel="modulepreload" crossorigin href="/assets/js/vendor-state-BrQ-Wpjh.js">
    <link rel="stylesheet" crossorigin href="/assets/css/index-CAyraQC6.css">
  </head>
  <body>
    <div id="root"></div>
  </body>
</html>
  ✅ :3001 → <a href="/login">Found</a>.
  ❌ :6831 → UNREACHABLE
  ❌ :6832 → UNREACHABLE
  ✅ :8000 → {"status":"healthy","service":"api-gateway"}
  ✅ :8002 → {"status":"healthy","service":"registry-service"}
  ✅ :8003 → {"status":"healthy","service":"job-service"}
  ✅ :8004 → {"status":"healthy","service":"storage-service"}
  ✅ :8006 → {"status":"healthy","service":"analytics-service"}
  ✅ :8007 → {"status":"healthy","service":"organization-service"}
  ✅ :8008 → {"service":"compliance-service","status":"healthy"}
  ✅ :8010 → {"service":"policy-service","status":"healthy"}
  ✅ :8011 → {"service":"m365-integration-service","status":"healthy"}
  ❌ :9090 → UNREACHABLE
  ❌ :9093 → UNREACHABLE
  ✅ :9101 → {"status":"healthy","service":"auth-service"}
  ✅ :9102 → {"status":"healthy","service":"registry-service"}
  ❌ :9103 → UNREACHABLE
  ❌ :9104 → UNREACHABLE
  ❌ :9105 → UNREACHABLE
  ❌ :9106 → UNREACHABLE
  ❌ :9107 → UNREACHABLE
  ❌ :9108 → UNREACHABLE
  ❌ :9109 → UNREACHABLE
  ❌ :9110 → UNREACHABLE
  ❌ :9111 → UNREACHABLE
```
## Test Results
```
FAIL	github.com/openprint/openprint/services/job-service/repository	6.458s
?   	github.com/openprint/openprint/services/m365-integration-service	[no test files]
?   	github.com/openprint/openprint/services/notification-service	[no test files]
ok  	github.com/openprint/openprint/services/notification-service/websocket	0.024s
?   	github.com/openprint/openprint/services/organization-service	[no test files]
?   	github.com/openprint/openprint/services/organization-service/handler	[no test files]
?   	github.com/openprint/openprint/services/organization-service/repository	[no test files]
2026/03/06 17:18:11 main_test.go:25: TestMain: Starting test database setup...
2026/03/06 17:18:16 main_test.go:30: Failed to setup test database: create postgres container: create container: container create: Error response from daemon: container create: creating container storage: the container name "reaper_acf6047c834c3805e7e97e7eb9346d1353d49e8a90232765c9365b12e76f41ec" is already in use by 61e52db47c7b027cbec4e089e242c47549abb2c3081b95a657bed18871111bfc. You have to remove that container to be able to reuse that name: that name is already in use: creating reaper failed
FAIL	github.com/openprint/openprint/services/policy-service	5.236s
?   	github.com/openprint/openprint/services/registry-service	[no test files]
?   	github.com/openprint/openprint/services/registry-service/handler	[no test files]
?   	github.com/openprint/openprint/services/registry-service/handlers	[no test files]
ok  	github.com/openprint/openprint/services/registry-service/repository	0.023s
?   	github.com/openprint/openprint/services/storage-service	[no test files]
ok  	github.com/openprint/openprint/services/storage-service/handler	0.021s
?   	github.com/openprint/openprint/services/storage-service/handlers	[no test files]
ok  	github.com/openprint/openprint/services/storage-service/storage	0.281s
ok  	github.com/openprint/openprint/tests/testutil	0.207s
FAIL
```
