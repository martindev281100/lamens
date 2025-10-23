# Prompt MVP (v2) – Công cụ tạo & deploy app lên K8s qua Argo CD với luồng **chọn Project → Repo → Path**

Bạn là **trợ lý DevOps**. Xây dựng **công cụ CLI** (hoặc TUI) giúp người dùng tạo và triển khai app lên Kubernetes với **2 môi trường** `staging` và `production`, sử dụng **Argo CD** + **Helm**. Công cụ cần:

---

## 1) Kết nối & khám phá Argo CD

- Hỏi đầu vào: `ARGOCD_SERVER`, `ARGOCD_AUTH` (token hoặc username/password). Kiểm thử kết nối.
- **Liệt kê danh sách Argo CD Projects** (API: `/api/v1/projects`). Hiển thị tên + mô tả, cho phép chọn **1 project**.
- Sau khi chọn Project, lấy danh sách **repos được phép** từ `spec.sourceRepos` của Project, đồng thời cho phép nhập tay **Git repo URL** nếu cần (ví dụ repo `launchpad`).

## 2) Chọn Repo & Path (Helm chart)

- Với repo đã chọn, công cụ **clone/fetch** ở `targetRevision` (branch/tag/commit) do user nhập hoặc mặc định `main`.
- **Quét các thư mục** có chứa `Chart.yaml` (ví dụ: `apps/api`, `apps/ai`, `services/foo`). Liệt kê **danh sách Path** để người dùng chọn.
- Sau khi chọn Path, nếu tồn tại `values.yaml`, **đọc và hiển thị preview** (ẩn giá trị nhạy cảm). Cho phép override nhanh các key phổ biến.

## 3) Hỏi & chuẩn hoá đầu vào triển khai

- **App name**: ví dụ `pigeonmail-ai-svc`.
- **Môi trường**: chọn `staging`, `production` hoặc **cả hai** → sẽ tạo 2 Argo CD Applications.
- **Namespace**: mặc định `<app>-<env>`. Nếu chưa tồn tại, **tự tạo**.
- **Target revision**: branch/tag/commit cho từng môi trường.
- **Cluster/Server** đích (địa chỉ API) cho từng môi trường (nếu khác nhau).
- **Docker Registry pull**:

  - Hỏi `registryServer`, `registryUsername`, `registryPassword`, `registryEmail` (tùy chọn).
  - **Tạo Secret** kiểu `kubernetes.io/dockerconfigjson` tên mặc định `docker` trong từng namespace **trước** khi tạo Application.

- **Uptrace/OTEL** (bật/tắt):

  - `OTEL_SERVICE_NAME` (mặc định `<app>-<env>`)
  - `OTEL_EXPORTER_OTLP_ENDPOINT` – **tự động phát hiện** `Service` Uptrace trong namespace do người dùng chỉ định (mặc định: `uptrace`). Quy tắc tìm: ưu tiên tên/label khớp `uptrace-collector`, `otel-collector`, hoặc label `app=uptrace`; chọn port **`4318` (HTTP OTLP)** nếu có, ngược lại dùng **`4317` (gRPC OTLP)**. Endpoint chuẩn: `http://<service>.<namespace>.svc:4318` (hoặc `grpc://<service>.<namespace>.svc:4317`).
  - **Uptrace Projects (ConfigMap)** – Liệt kê các project có sẵn từ **ConfigMap** trong namespace Uptrace (ví dụ `uptrace-projects`, key `projects.yaml|projects.json`). Cho phép người dùng **tạo project mới qua form** (các trường gợi ý: `name`, `dsn`, `sampleRate`, `servicePrefix`, `attrs`…), sau đó **ghi/patch** vào ConfigMap; nếu ConfigMap chưa tồn tại thì **tạo mới**. Khi người dùng chọn project, tự động điền `OTEL_EXPORTER_OTLP_HEADERS` với `uptrace-dsn=<dsn>` và (nếu có) gợi ý tiền tố cho `OTEL_SERVICE_NAME` theo `servicePrefix`.
  - `OTEL_EXPORTER_OTLP_HEADERS` – tự lấy `uptrace-dsn=…` từ **Secret** trong namespace Uptrace (mặc định: secret `uptrace-env` với key `UPTRACE_DSN`) **hoặc** từ project người dùng đã chọn trong ConfigMap. Nếu không tìm thấy → hỏi người dùng nhập thủ công.

- **ENV/Secrets app**: key=value + `envSecret` để map vào chart (nếu chart hỗ trợ).
- **Ingress/replicas**: cho phép nhập nhanh.

## 4) Ánh xạ vào Helm values

- Công cụ hợp nhất `values.yaml` gốc với **overrides theo môi trường**; đảm bảo có:

  - `image.repository`, `image.tag` (cho phép **trống ở production** nếu dùng promotion tự động)
  - `imagePullSecrets: [{ name: docker }]`
  - `envSecret: <app>-<env>-secret` (hoặc tên người dùng nhập)
  - `podLabels.app = <app>`, `podLabels.version = <tag>` (nếu có)
  - Thêm ENV Uptrace nếu bật.

## 5) Tạo & Sync Argo CD Application

- **Render YAML** cho mỗi môi trường và **áp dụng** qua Argo CD (hoặc `kubectl apply -f -`).
- Bật `syncPolicy.automated` (`prune`, `selfHeal`).
- **Sync** và theo dõi trạng thái `Healthy/Synced`. Thất bại → in lỗi và gợi ý rollback.

## 6) Ví dụ `values.yaml` (chuẩn hoá theo chart dạng `ght-app`)

```yaml
ght-app:
  replicaCount: 1
  port: 5005
  image:
    repository: ethannguyen98/ght-pigeonmail-ai-svc
    tag: v0.0.2
  imagePullSecrets:
    - name: docker
  envSecret: ght-pigeonmail-ai-svc-secret
  podLabels:
    app: ght-summarify-ai-svc
    version: v0.0.2
  # (tuỳ chọn) env:
  #   - name: OTEL_SERVICE_NAME
  #     value: ght-pigeonmail-ai-svc-staging
  #   - name: OTEL_EXPORTER_OTLP_ENDPOINT
  #     value: http://<uptrace-svc>.<uptrace-ns>.svc:4318
  #   - name: OTEL_EXPORTER_OTLP_HEADERS
  #     value: uptrace-dsn=xxxxx
```

> Lưu ý: nếu chart dùng khoá khác (ví dụ `env` thay vì `envSecret`), công cụ cần **map linh hoạt**.

## 7) Mẫu Argo CD Application YAML (rút gọn)

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: <app>-<env>
  namespace: argocd
spec:
  project: <selected-argocd-project>
  source:
    repoURL: <selected-repo-url>
    path: <selected-path>
    targetRevision: <branch-or-tag>
    helm:
      valueFiles: ["values.yaml"]
      # hoặc inline overrides nếu muốn
      # values: |
      #   ght-app:
      #     imagePullSecrets:
      #       - name: docker
  destination:
    server: <k8s-api-server>
    namespace: <app>-<env>
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

## 8) Tạo Secret docker trước khi tạo Application (YAML-only + kiểm tra tồn tại)

- Công cụ **chỉ hỗ trợ tạo Secret bằng YAML** với trường `.dockerconfigjson` ở dạng **base64**. _Không_ dùng `kubectl create secret docker-registry`.
- **Kiểm tra tồn tại trước khi hỏi đầu vào**:

  - Nếu Secret `docker` trong namespace **đã tồn tại** và `type` là `kubernetes.io/dockerconfigjson` → **bỏ qua bước hỏi** base64 và **không tạo lại**.
  - Nếu **chưa tồn tại** → yêu cầu người dùng cung cấp `BASE64_DOCKERCONFIGJSON` (nội dung `~/.docker/config.json` đã mã hoá base64, không xuống dòng) rồi áp dụng YAML bên dưới.

- Gợi ý kiểm tra tồn tại (tham khảo):

  - `kubectl -n <ns> get secret docker -o jsonpath='{.type}'` → mong đợi `kubernetes.io/dockerconfigjson`.

### YAML Secret (áp dụng khi chưa có)

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: docker
  namespace: <app>-<env>
type: kubernetes.io/dockerconfigjson
data:
  .dockerconfigjson: <BASE64_DOCKERCONFIGJSON>
```

**Lưu ý**

- `BASE64_DOCKERCONFIGJSON` là **toàn bộ** nội dung file `~/.docker/config.json` sau khi base64 (**không** thêm ký tự xuống dòng). Ví dụ tạo nhanh:

  - macOS/Linux: `cat ~/.docker/config.json | base64 | tr -d '\n'`

- Sau khi xác nhận Secret đã có, tiếp tục bước tạo Argo CD Application.

## 9) (Mới) Tạo **App ENV Secret** trong namespace ứng dụng (Opaque)

Sau khi đã có đủ thông tin Uptrace (endpoint, DSN hoặc project từ ConfigMap), công cụ sẽ **tạo một Secret Opaque chứa ENV** cho app trong **namespace mục tiêu**. Secret này sẽ được tham chiếu bởi chart (ví dụ khoá `envSecret`).

- **Tên mặc định**: `<app>-<env>-secret` (cho phép override, ví dụ: `ght-cms-dms-svc-secret`).
- **Namespace**: `<app>-<env>` (hoặc tuỳ người dùng chọn, ví dụ: `cms-staging`).
- **Khoá phổ biến** (base64 trong YAML):

  - `ENVIRONMENT` → `staging`/`production`.
  - `LOG_LEVEL` → mặc định `info` (cho phép nhập, ví dụ `error`).
  - `TRACING_ENDPOINT` → lấy từ khám phá Uptrace. Chấp nhận **dạng đầy đủ** (`http://svc.ns.svc:4318`/`grpc://...:4317`) hoặc **host:port** tuỳ chart; khuyến nghị giữ nguyên endpoint đã phát hiện.
  - `UPTRACE_DSN` → từ Uptrace Secret hoặc Uptrace Project trong ConfigMap.

### YAML mẫu (Opaque)

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: <app>-<env>-secret
  namespace: <app>-<env>
type: Opaque
data:
  ENVIRONMENT: <base64("staging")>
  LOG_LEVEL: <base64("info")>
  TRACING_ENDPOINT: <base64("http://uptrace-collector.uptrace.svc:4318")>
  UPTRACE_DSN: <base64("http://token@uptrace-collector.uptrace.svc:4318/2")>
```

> Ví dụ theo bạn cung cấp:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: ght-cms-dms-svc-secret
  namespace: cms-staging
data:
  ENVIRONMENT: c3RhZ2luZw==
  LOG_LEVEL: ZXJyb3I=
  TRACING_ENDPOINT: bXktdXB0cmFjZS51cHRyYWNlLnN2Yy5jbHVzdGVyLmxvY2FsOjE0MzE3
  UPTRACE_DSN: aHR0cDovL2Ntc0BteS11cHRyYWNlLnVwdHJhY2Uuc3ZjLmNsdXN0ZXIubG9jYWw6MTQzMTgvMg==
type: Opaque
```

**Quy tắc idempotent**

- Nếu Secret tên đó đã tồn tại trong namespace, **không hỏi lại**, cho phép **patch/merge** khi người dùng yêu cầu cập nhật một vài khoá.
- Cho phép tham số `--dry-run` để chỉ render YAML.

**Gắn vào chart**

- Với chart dạng `ght-app`, đảm bảo có tham chiếu tới secret này (ví dụ `.Values.envSecret`).
- Nếu chart dùng `.Values.env` (env trực tiếp), công cụ có thể gợi ý chuyển sang dùng secret hoặc tiêm trực tiếp.

## 9) Kiểm tra sau deploy

- Secret `docker` tồn tại: `kubectl -n <ns> get secret docker -o yaml | grep -E "type:|\.dockerconfigjson"`
- Pods **không** gặp `ImagePullBackOff`.
- `kubectl -n <ns> get application <app>-<env> -o jsonpath='{.status.health.status}{"/"}{.status.sync.status}'` → `Healthy/Synced`.
- Nếu bật Uptrace: metrics/traces xuất hiện với `service.name = <app>-<env>`.

## 10) Tiêu chí hoàn thành (MVP)

- UI chọn **Project → Repo → Path** hoạt động mượt; quét tối đa ~30 path có `Chart.yaml`.
- Tạo **namespace + docker pull Secret** trước khi tạo Application.
- Tạo 1–2 Applications (staging/prod), sync xong, pod chạy OK.
- Có **tóm tắt cấu hình** trước khi apply, ghi log lỗi rõ ràng và gợi ý rollback.

---

# 11) Triển khai công cụ **bằng Go** (stack, snippet, pseudo-code)

## Ngăn xếp & thư viện đề xuất (Go)

- **CLI/TUI**: `github.com/spf13/cobra`, `github.com/AlecAivazis/survey/v2`
- **Kubernetes client**: `k8s.io/client-go`, `k8s.io/api/...`, `k8s.io/apimachinery/...`
- **Dynamic apply YAML**: `sigs.k8s.io/controller-runtime/pkg/client` (tuỳ chọn) hoặc parse YAML → đối tượng Go
- **YAML/Templating**: `sigs.k8s.io/yaml`, `text/template`
- **Git**: `github.com/go-git/go-git/v5` (hoặc gọi `git` local)
- **FS scan**: `filepath.WalkDir`

## Pseudo-code tổng quan (Go)

```go
func run() error {
  cfg := loadKubeConfig() // KUBECONFIG hoặc in-cluster
  k8s := kubernetes.NewForConfigOrDie(cfg)
  dyn := dynamic.NewForConfigOrDie(cfg)

  // 1) Liệt kê AppProject (Argo CD) để chọn
  projects := listArgoCDProjects(dyn, "argocd")
  project := promptSelect(projects)

  // 2) Chọn repo & path
  repoURL := promptRepoURL(project)
  rev := promptTargetRevision()
  workdir := cloneOrFetch(repoURL, rev)
  paths := findChartPaths(workdir, 30) // quét folder có Chart.yaml
  chartPath := promptSelect(paths)

  // 3) Nhập app/env/ns + Uptrace + overrides
  app := promptAppName()
  envs := promptEnvs(["staging", "production"]) // chọn 1 hoặc 2

  // Hỏi namespace Uptrace (mặc định: "uptrace") để tự động phát hiện endpoint + headers
  uptraceNS := promptDefault("uptrace")

  for _, env := range envs {
    ns := defaultNS(app, env)

    // 3a) Đảm bảo namespace
    ensureNamespace(k8s, ns)

    // 3b) Docker Secret YAML-only: check tồn tại, nếu chưa thì yêu cầu base64
    if !existsDockerSecret(k8s, ns, "docker") {
      b64 := promptBase64DockerConfigJSON()
      applyDockerSecretYAML(k8s, ns, b64)
    }

    // 3c) Khám phá Uptrace: tìm Service + DSN
    endpoint, headers := discoverUptrace(k8s, uptraceNS)

    // 3d) Render Argo CD Application YAML
    appYAML := renderApplicationYAML(ApplicationInput{
      Name: app + "-" + env,
      Project: project,
      RepoURL: repoURL,
      Path: chartPath,
      TargetRevision: rev,
      Namespace: ns,
      InlineValues: buildInlineValues(app, env, endpoint, headers),
    })

    // 3e) Apply Application (CRD argoproj.io/v1alpha1)
    applyYAML(dyn, appYAML, "argocd")
  }

  printSummary()
  return nil
}
```

## Liệt kê **Argo CD Projects** qua K8s CRD (Go)

> AppProject là CRD của Argo CD (thường nằm **namespace `argocd`**). Không cần gọi REST của Argo CD.

```go
var (
  appProjectGVR = schema.GroupVersionResource{
    Group:    "argoproj.io",
    Version:  "v1alpha1",
    Resource: "appprojects",
  }
)

func listArgoCDProjects(dyn dynamic.Interface, ns string) []string {
  list, err := dyn.Resource(appProjectGVR).Namespace(ns).List(context.TODO(), metav1.ListOptions{})
  if err != nil { panic(err) }
  out := make([]string, 0, len(list.Items))
  for _, it := range list.Items {
    out = append(out, it.GetName())
  }
  sort.Strings(out)
  return out
}
```

## Kiểm tra & tạo **Namespace** (Go)

```go
func ensureNamespace(k8s *kubernetes.Clientset, ns string) {
  if _, err := k8s.CoreV1().Namespaces().Get(context.TODO(), ns, metav1.GetOptions{}); err == nil {
    return
  }
  _, err := k8s.CoreV1().Namespaces().Create(context.TODO(), &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}, metav1.CreateOptions{})
  if err != nil { panic(err) }
}
```

## Kiểm tra tồn tại & tạo **docker Secret** (YAML-only) (Go)

```go
func existsDockerSecret(k8s *kubernetes.Clientset, ns, name string) bool {
  s, err := k8s.CoreV1().Secrets(ns).Get(context.TODO(), name, metav1.GetOptions{})
  if err != nil { return false }
  return s.Type == v1.SecretTypeDockerConfigJson
}

func applyDockerSecretYAML(k8s *kubernetes.Clientset, ns, b64 string) {
  // Build object từ Go struct
  sec := &v1.Secret{
    ObjectMeta: metav1.ObjectMeta{Name: "docker", Namespace: ns},
    Type:       v1.SecretTypeDockerConfigJson,
    Data:       map[string][]byte{ ".dockerconfigjson": []byte(b64) }, // b64 string giữ nguyên
  }
  // Lưu ý: Data kỳ vọng **raw bytes** đã base64-encoded ở YAML; nếu tạo từ struct, dùng `StringData` để tránh double-base64
  sec.StringData = map[string]string{".dockerconfigjson": b64}
  // Ưu tiên dùng StringData để server tự base64 hoá
  sec.Data = nil

  if _, err := k8s.CoreV1().Secrets(ns).Create(context.TODO(), sec, metav1.CreateOptions{}); err != nil {
    panic(err)
  }
}
```

> **Chú ý**: Nếu bạn render YAML thủ công, đặt `type: kubernetes.io/dockerconfigjson` và trường `data..dockerconfigjson` là **base64**.

## Quét repo & chọn **path có Chart.yaml** (Go)

```go
func findChartPaths(root string, limit int) []string {
  var out []string
  filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
    if err != nil || d.IsDir() == false { return nil }
    chart := filepath.Join(p, "Chart.yaml")
    if _, e := os.Stat(chart); e == nil {
      out = append(out, strings.TrimPrefix(p, root+string(os.PathSeparator)))
      if len(out) >= limit { return io.EOF } // stop early
    }
    return nil
  })
  sort.Strings(out)
  return out
}
```

## Khám phá **Uptrace Service** & DSN (Go)

```go
func discoverUptrace(k8s *kubernetes.Clientset, ns string) (endpoint string, headers map[string]string) {
  // 1) Tìm Service ưu tiên theo tên/label
  svcs, _ := k8s.CoreV1().Services(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=uptrace"})
  var svc *v1.Service
  for i := range svcs.Items { svc = &svcs.Items[i]; break }
  // fallback theo tên phổ biến
  if svc == nil {
    names := []string{"uptrace-collector", "otel-collector", "uptrace"}
    for _, n := range names {
      s, err := k8s.CoreV1().Services(ns).Get(context.TODO(), n, metav1.GetOptions{})
      if err == nil { svc = s; break }
    }
  }
  // 2) Chọn port: ưu tiên 4318 (HTTP), rồi 4317 (gRPC), rồi theo port name có "otlp"
  port := int32(0)
  if svc != nil {
    for _, p := range svc.Spec.Ports { if p.Port == 4318 { port = p.Port; break } }
    if port == 0 { for _, p := range svc.Spec.Ports { if p.Port == 4317 { port = p.Port; break } } }
    if port == 0 { for _, p := range svc.Spec.Ports { if strings.Contains(strings.ToLower(p.Name), "otlp") { port = p.Port; break } } }
  }
  host := ""
  if svc != nil && port != 0 { host = fmt.Sprintf("%s.%s.svc", svc.Name, ns) }
  // 3) Lập endpoint
  if host != "" {
    if port == 4317 { endpoint = fmt.Sprintf("grpc://%s:%d", host, port) } else { endpoint = fmt.Sprintf("http://%s:%d", host, port) }
  }
  // 4) Lấy DSN từ Secret mặc định
  headers = map[string]string{}
  if sec, err := k8s.CoreV1().Secrets(ns).Get(context.TODO(), "uptrace-env", metav1.GetOptions{}); err == nil {
    if b, ok := sec.Data["UPTRACE_DSN"]; ok && len(b) > 0 {
      headers["uptrace-dsn"] = string(b)
    }
  }
  return
}
```

## Tạo **Argo CD Application** bằng Go (render YAML)

```go
type ApplicationInput struct {
  Name, Project, RepoURL, Path, TargetRevision, Namespace string
  InlineValues string // optional inline Helm values
}

const appTpl = `apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: {{.Name}}
  namespace: argocd
spec:
  project: {{.Project}}
  source:
    repoURL: {{.RepoURL}}
    path: {{.Path}}
    targetRevision: {{.TargetRevision}}
    helm:
      valueFiles: ["values.yaml"]
{{- if .InlineValues }}
      values: |
{{ .InlineValues | indent 8 }}
{{- end }}
  destination:
    server: https://kubernetes.default.svc
    namespace: {{.Namespace}}
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
`
```

Hàm render & apply:

```go
func renderApplicationYAML(in ApplicationInput) []byte {
  var buf bytes.Buffer
  tpl := template.Must(template.New("app").Funcs(template.FuncMap{
    "indent": func(n int, s string) string { pad := strings.Repeat(" ", n); return pad + strings.ReplaceAll(s, "\n", "\n"+pad) },
  }).Parse(appTpl))
  if err := tpl.Execute(&buf, in); err != nil { panic(err) }
  return buf.Bytes()
}

func applyYAML(dyn dynamic.Interface, yml []byte, ns string) {
  // Parse YAML -> Unstructured rồi Create/Apply
  dec := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(yml), 4096)
  for {
    var obj unstructured.Unstructured
    if err := dec.Decode(&obj); err != nil { if err == io.EOF { break }; panic(err) }
    gvk := obj.GroupVersionKind()
    m, _ := meta.UnsafeGuessKindToResource(gvk)
    ri := dyn.Resource(m).Namespace(obj.GetNamespace())
    // Try Create, if exists then Update/Patch
    if _, err := ri.Create(context.TODO(), &obj, metav1.CreateOptions{}); apierrors.IsAlreadyExists(err) {
      current, _ := ri.Get(context.TODO(), obj.GetName(), metav1.GetOptions{})
      obj.SetResourceVersion(current.GetResourceVersion())
      if _, err := ri.Update(context.TODO(), &obj, metav1.UpdateOptions{}); err != nil { panic(err) }
    } else if err != nil { panic(err) }
  }
}
```

## Build inline Helm values từ `values.yaml` mẫu `ght-app`

```go
func buildInlineValues(app, env, endpoint string, headers map[string]string) string {
  lines := []string{
    "ght-app:",
    "  imagePullSecrets:",
    "    - name: docker",
    fmt.Sprintf("  podLabels:\n    app: %s", app),
  }
  if endpoint != "" {
    lines = append(lines, fmt.Sprintf("  env:\n    - name: OTEL_EXPORTER_OTLP_ENDPOINT\n      value: %s", endpoint))
  }
  if len(headers) > 0 {
    // chỉ lấy uptrace-dsn
    if dsn, ok := headers["uptrace-dsn"]; ok {
      lines = append(lines, fmt.Sprintf("    - name: OTEL_EXPORTER_OTLP_HEADERS\n      value: uptrace-dsn=%s", dsn))
    }
  }
  return strings.Join(lines, "\n") + "\n"
}
```

---

**Gợi ý đóng gói**: build 1 binary `kargo-bootstrap` (ví dụ), nhận tham số `--server`, `--kubeconfig`, `--argocd-ns=argocd`, `--repo`, `--rev`, `--envs=staging,production`, `--non-interactive` để phù hợp CI.

---

# 12) Nâng cấp thành **MCP Server** (Go) cho quy trình tạo & deploy Argo CD

> Mục tiêu: đóng gói toàn bộ luồng **Project → Repo → Path → Namespace/Secret → Application → Sync** thành **MCP Server** để IDE/assistant có thể gọi qua **tools**.

## 12.1 Kiến trúc & Transport

- **Protocol**: Model Context Protocol (MCP), JSON-RPC.
- **Transport**: `stdio` (mặc định) hoặc `tcp` nếu cần chạy tách rời.
- **Runtime**: Go 1.22+.
- **Auth/Secrets**: lấy từ biến môi trường hoặc file cấu hình (kubeconfig, Argo CD token). Không truyền secrets qua tool params nếu tránh được.

## 12.2 Danh sách Tools (MCP)

Các tool tối giản để ghép thành pipeline:

1. `list_projects`

   - **desc**: Liệt kê AppProjects (CRD) trong namespace Argo CD.
   - **params**: `{ "namespace": "argocd" }`
   - **result**: `{ "projects": ["staging", "production", ...] }`

2. `list_repo_paths`

   - **desc**: Clone/fetch repo ở `targetRevision`, quét thư mục có `Chart.yaml`.
   - **params**: `{ "repoUrl": string, "targetRevision": string, "limit": number }`
   - **result**: `{ "paths": ["apps/api", "apps/ai", ...] }`

3. `ensure_namespace`

   - **params**: `{ "namespace": string }`
   - **result**: `{ "created": bool }`

4. `upsert_docker_secret_yaml`

   - **desc**: **YAML-only** tạo `kubernetes.io/dockerconfigjson`; nếu đã tồn tại & type đúng thì bỏ qua.
   - **params**: `{ "namespace": string, "name": "docker", "dockerconfigjsonB64": string }`
   - **result**: `{ "created": bool, "skipped": bool }`

5. `discover_uptrace`

   - **desc**: Tìm `Service` Uptrace (port 4318/4317), đọc Secret DSN **và** liệt kê/ghi **Uptrace Projects** từ ConfigMap.
   - **params**: `{ "namespace": "uptrace", "serviceNames?": ["uptrace-collector","otel-collector","uptrace"], "secretName?": "uptrace-env", "dsnKey?": "UPTRACE_DSN", "configMapName?": "uptrace-projects", "configKey?": "projects.yaml" }`
   - **result**: `{ "endpoint": string, "headers": {"uptrace-dsn": string}, "projects": [ {"name": string, "dsn": string, ...} ], "service": string, "port": number, "found": bool }`

6. `upsert_uptrace_project`

   - **desc**: Tạo/cập nhật một project trong ConfigMap (merge theo name).
   - **params**: `{ "namespace": "uptrace", "configMapName?": "uptrace-projects", "configKey?": "projects.yaml", "project": {"name": string, "dsn": string, "sampleRate?": number, "servicePrefix?": string, "attrs?": map} }`
   - **result**: `{ "updated": bool }`

7. `upsert_app_env_secret`

   - **desc**: Tạo/cập nhật **Opaque Secret** ENV trong namespace ứng dụng, sử dụng thông tin từ Uptrace (endpoint, dsn/project) và các trường người dùng nhập.
   - **params**: `{ "namespace": string, "name": string, "environment": "staging|production|...", "logLevel": string, "tracingEndpoint": string, "uptraceDsn": string, "merge": bool }`
   - **result**: `{ "created": bool, "updated": bool }`

8. `render_application_yaml`

   - **desc**: Render YAML Argo CD Application theo `ApplicationInput` (mục 11).
   - **params**: `{ Name, Project, RepoURL, Path, TargetRevision, Namespace, InlineValues? }`
   - **result**: `{ "yaml": string }`

9. `apply_yaml`

   - **desc**: Apply 1 hoặc nhiều tài nguyên YAML (CRDs, Secret, Application…).
   - **params**: `{ "yaml": string, "defaultNamespace?": string }`
   - **result**: `{ "applied": [ {"kind": string, "name": string, "namespace": string} ] }`

10. `sync_application`

- **desc**: Gọi sync (nếu dùng Argo CD API) **hoặc** chờ controller reconcile nếu chỉ apply CRD.
- **params**: `{ "name": string, "namespace": "argocd", "timeoutSeconds?": number }`
- **result**: `{ "status": "Synced|OutOfSync", "health": "Healthy|Degraded", "message?": string }`

11. `preview_values`

- **desc**: Đọc `values.yaml` ở path đã chọn, trả về phần xem trước (mask secrets).
- **params**: `{ "repoUrl": string, "targetRevision": string, "path": string, "maxBytes?": number }`
- **result**: `{ "preview": string }`

## 12.3 Khai báo Server Capabilities (manifest tối giản) (manifest tối giản)

```json
{
  "name": "kargo-argocd-bootstrap",
  "version": "0.1.0",
  "capabilities": {
    "tools": [
      "list_projects",
      "list_repo_paths",
      "ensure_namespace",
      "upsert_docker_secret_yaml",
      "render_application_yaml",
      "apply_yaml",
      "sync_application",
      "preview_values",
      "discover_uptrace"
    ]
  }
}
```

## 12.4 Cấu hình MCP Client (ví dụ `clients.json` cho IDE/assistant)

```json
{
  "mcpServers": {
    "kargo-argocd-bootstrap": {
      "command": "/usr/local/bin/kargo-bootstrap-mcp",
      "args": [],
      "env": {
        "KUBECONFIG": "/home/user/.kube/config",
        "ARGOCD_NAMESPACE": "argocd",
        "ARGOCD_SERVER": "https://argocd.example.com",
        "ARGOCD_AUTH_TOKEN": "<token>"
      }
    }
  }
}
```

## 12.5 Khung mã **Go MCP Server** (skeleton)

> Sử dụng một SDK MCP cho Go (ví dụ `go-mcp`). Tên gói có thể khác nhau tuỳ thời điểm; bên dưới minh hoạ interface chung.

```go
package main

import (
  "context"
  "log"
  "os"

  mcp "github.com/modelcontextprotocol/go-mcp" // ví dụ SDK
)

func main() {
  srv := mcp.NewServer(mcp.ServerOptions{ Name: "kargo-argocd-bootstrap", Version: "0.1.0" })

  // Đăng ký tools
  srv.Tool("list_projects", listProjects)
  srv.Tool("list_repo_paths", listRepoPaths)
  srv.Tool("ensure_namespace", ensureNamespaceTool)
  srv.Tool("upsert_docker_secret_yaml", upsertDockerSecretYAMLTool)
  srv.Tool("render_application_yaml", renderApplicationYAMLTool)
  srv.Tool("apply_yaml", applyYAMLTool)
  srv.Tool("sync_application", syncApplicationTool)
  srv.Tool("preview_values", previewValuesTool)
  srv.Tool("discover_uptrace", discoverUptraceTool)
  srv.Tool("upsert_uptrace_project", upsertUptraceProjectTool)
  srv.Tool("upsert_app_env_secret", upsertAppEnvSecretTool)

  // Bắt đầu stdio transport
  if err := srv.ServeSTDIO(context.Background()); err != nil {
    log.Fatal(err)
  }
}
```

### Triển khai handlers (rút gọn, tái dùng hàm ở mục 11)

```go
func listProjects(ctx context.Context, req mcp.ToolRequest) (any, error) {
  ns := getenvDefault("ARGOCD_NAMESPACE", "argocd")
  dyn := mustDynamic()
  names := listArgoCDProjects(dyn, ns)
  return map[string]any{"projects": names}, nil
}

func listRepoPaths(ctx context.Context, req mcp.ToolRequest) (any, error) {
  var p struct{ RepoURL, TargetRevision string; Limit int }
  if err := req.Params(&p); err != nil { return nil, err }
  dir := cloneOrFetch(p.RepoURL, orDefault(p.TargetRevision, "main"))
  paths := findChartPaths(dir, ifZero(p.Limit, 30))
  return map[string]any{"paths": paths}, nil
}

func ensureNamespaceTool(ctx context.Context, req mcp.ToolRequest) (any, error) {
  var p struct{ Namespace string }
  if err := req.Params(&p); err != nil { return nil, err }
  k8s := mustClientset()
  created := ensureNamespaceReturn(k8s, p.Namespace)
  return map[string]any{"created": created}, nil
}

func upsertDockerSecretYAMLTool(ctx context.Context, req mcp.ToolRequest) (any, error) {
  var p struct{ Namespace, Name, DockerconfigjsonB64 string }
  if err := req.Params(&p); err != nil { return nil, err }
  k8s := mustClientset()
  if existsDockerSecret(k8s, p.Namespace, p.Name) {
    return map[string]any{"created": false, "skipped": true}, nil
  }
  applyDockerSecretYAML(k8s, p.Namespace, p.DockerconfigjsonB64)
  return map[string]any{"created": true, "skipped": false}, nil
}

func renderApplicationYAMLTool(ctx context.Context, req mcp.ToolRequest) (any, error) {
  var in ApplicationInput
  if err := req.Params(&in); err != nil { return nil, err }
  y := renderApplicationYAML(in)
  return map[string]any{"yaml": string(y)}, nil
}

func applyYAMLTool(ctx context.Context, req mcp.ToolRequest) (any, error) {
  var p struct{ Yaml, DefaultNamespace string }
  if err := req.Params(&p); err != nil { return nil, err }
  dyn := mustDynamic()
  applied := applyYAMLWithResult(dyn, []byte(p.Yaml), p.DefaultNamespace)
  return map[string]any{"applied": applied}, nil
}

func syncApplicationTool(ctx context.Context, req mcp.ToolRequest) (any, error) {
  var p struct{ Name, Namespace string; TimeoutSeconds int }
  if err := req.Params(&p); err != nil { return nil, err }
  // Tuỳ chọn: gọi Argo CD API để Sync; nếu không, poll status CR
  st := waitForAppStatus(p.Namespace, p.Name, ifZero(p.TimeoutSeconds, 180))
  return st, nil
}

func previewValuesTool(ctx context.Context, req mcp.ToolRequest) (any, error) {
  var p struct{ RepoURL, TargetRevision, Path string; MaxBytes int }
  if err := req.Params(&p); err != nil { return nil, err }
  dir := cloneOrFetch(p.RepoURL, orDefault(p.TargetRevision, "main"))
  raw := readFileLimit(filepath.Join(dir, p.Path, "values.yaml"), ifZero(p.MaxBytes, 64*1024))
  return map[string]any{"preview": maskSecrets(string(raw))}, nil
}

func discoverUptraceTool(ctx context.Context, req mcp.ToolRequest) (any, error) {
  var p struct{ Namespace, SecretName, DsnKey string; ServiceNames []string }
  if err := req.Params(&p); err != nil { return nil, err }
  if p.Namespace == "" { p.Namespace = "uptrace" }
  if p.SecretName == "" { p.SecretName = "uptrace-env" }
  if p.DsnKey == "" { p.DsnKey = "UPTRACE_DSN" }
  k8s := mustClientset()

  // Tìm service
  var svc *v1.Service
  if len(p.ServiceNames) > 0 {
    for _, n := range p.ServiceNames {
      if s, err := k8s.CoreV1().Services(p.Namespace).Get(context.TODO(), n, metav1.GetOptions{}); err == nil { svc = s; break }
    }
  }
  if svc == nil {
    lst, _ := k8s.CoreV1().Services(p.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=uptrace"})
    if len(lst.Items) > 0 { svc = &lst.Items[0] }
  }
  if svc == nil {
    return map[string]any{"found": false}, nil
  }
  // Port logic
  port := int32(0)
  for _, pr := range svc.Spec.Ports { if pr.Port == 4318 { port = pr.Port; break } }
  if port == 0 { for _, pr := range svc.Spec.Ports { if pr.Port == 4317 { port = pr.Port; break } } }
  if port == 0 { for _, pr := range svc.Spec.Ports { if strings.Contains(strings.ToLower(pr.Name), "otlp") { port = pr.Port; break } } }
  host := fmt.Sprintf("%s.%s.svc", svc.Name, p.Namespace)
  endpoint := fmt.Sprintf("http://%s:%d", host, port)
  if port == 4317 { endpoint = fmt.Sprintf("grpc://%s:%d", host, port) }

  // Headers/DSN
  headers := map[string]string{}
  if sec, err := k8s.CoreV1().Secrets(p.Namespace).Get(context.TODO(), p.SecretName, metav1.GetOptions{}); err == nil {
    if b, ok := sec.Data[p.DsnKey]; ok && len(b) > 0 { headers["uptrace-dsn"] = string(b) }
  }
  return map[string]any{"found": true, "service": svc.Name, "port": port, "endpoint": endpoint, "headers": headers}, nil
}
```

## 12.6 Security & RBAC

- Service account chạy MCP server cần quyền:

  - `get/list` AppProjects (CRD `argoproj.io/appprojects`) trong `argocd`.
  - `get/create/update` `secrets` trong namespace đích.
  - `create/update` `applications.argoproj.io` trong `argocd`.
  - `get/list` `services` và `secrets` **trong namespace Uptrace** (ví dụ `uptrace`) để tự động phát hiện endpoint/DSN.

- Không lưu trữ `.dockerconfigjson` plaintext; chỉ nhận **base64** và apply trực tiếp.

## 12.7 UX gợi ý (templates cho Assistant)

- Prompt chain mẫu:

  1. Gọi `list_projects` → người dùng chọn project.
  2. Nhập `repoUrl`, `targetRevision` → `list_repo_paths` → chọn path.
  3. `ensure_namespace` với `<app>-<env>`.
  4. `discover_uptrace` với namespace (mặc định `uptrace`) → nhận `endpoint` + `headers`.
  5. `upsert_docker_secret_yaml` (chỉ khi thiếu).
  6. `render_application_yaml` (truyền inline values từ bước 4) → hiển thị YAML → xác nhận.
  7. `apply_yaml` → `sync_application` → trả trạng thái.

## 12.8 Packaging/Deploy

- Xuất binary: `kargo-bootstrap-mcp`.
- Cấu hình client MCP bằng `clients.json` (ở trên) trong IDE/assistant.
- Container image (tuỳ chọn) gồm: binary + `kubectl`/`git` (nếu cần gọi ngoài) hoặc dùng thư viện Go thuần.

## 12.9 Testing nhanh

- Unit: mock `client-go` & `dynamic.Interface` cho apply/list.
- E2E: kind cluster + Argo CD + repo demo có `Chart.yaml`.
- Case bắt buộc: Secret đã tồn tại (skip), lỗi ImagePullBackOff do thiếu `imagePullSecrets`, sync thất bại, **không tìm thấy Uptrace** (phải cho phép nhập thủ công endpoint/DSN), **ENV Secret** được tạo/cập nhật đúng với giá trị base64.

---

### Handlers & helpers mới (Go)

```go
func upsertUptraceProjectTool(ctx context.Context, req mcp.ToolRequest) (any, error) {
  var p struct{ Namespace, ConfigMapName, ConfigKey string; Project map[string]any }
  if err := req.Params(&p); err != nil { return nil, err }
  if p.Namespace == "" { p.Namespace = "uptrace" }
  if p.ConfigMapName == "" { p.ConfigMapName = "uptrace-projects" }
  if p.ConfigKey == "" { p.ConfigKey = "projects.yaml" }
  k8s := mustClientset()
  updated, err := upsertProjectToConfigMap(k8s, p.Namespace, p.ConfigMapName, p.ConfigKey, p.Project)
  if err != nil { return nil, err }
  return map[string]any{"updated": updated}, nil
}

func upsertAppEnvSecretTool(ctx context.Context, req mcp.ToolRequest) (any, error) {
  var p struct{
    Namespace, Name string
    Environment string
    LogLevel string
    TracingEndpoint string
    UptraceDsn string
    Merge bool
  }
  if err := req.Params(&p); err != nil { return nil, err }
  k8s := mustClientset()
  created, updated, err := upsertOpaqueEnvSecret(k8s, p.Namespace, p.Name, map[string]string{
    "ENVIRONMENT": p.Environment,
    "LOG_LEVEL": p.LogLevel,
    "TRACING_ENDPOINT": p.TracingEndpoint,
    "UPTRACE_DSN": p.UptraceDsn,
  }, p.Merge)
  if err != nil { return nil, err }
  return map[string]any{"created": created, "updated": updated}, nil
}

func upsertProjectToConfigMap(k8s *kubernetes.Clientset, ns, cmName, key string, proj map[string]any) (bool, error) {
  cm, err := k8s.CoreV1().ConfigMaps(ns).Get(context.TODO(), cmName, metav1.GetOptions{})
  if apierrors.IsNotFound(err) {
    dataBytes, _ := yaml.Marshal(map[string]any{"projects": []any{proj}})
    _, err = k8s.CoreV1().ConfigMaps(ns).Create(context.TODO(), &v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: cmName}, Data: map[string]string{key: string(dataBytes)}}, metav1.CreateOptions{})
    return true, err
  } else if err != nil { return false, err }
  raw := cm.Data[key]
  list := parseProjects(raw) // []map[string]any
  // merge theo name
  name := fmt.Sprint(proj["name"])
  found := false
  for i := range list { if fmt.Sprint(list[i]["name"]) == name { list[i] = proj; found = true; break } }
  if !found { list = append(list, proj) }
  out, _ := yaml.Marshal(map[string]any{"projects": list})
  cm.Data[key] = string(out)
  _, err = k8s.CoreV1().ConfigMaps(ns).Update(context.TODO(), cm, metav1.UpdateOptions{})
  return true, err
}

func parseProjects(raw string) []map[string]any {
  if strings.TrimSpace(raw) == "" { return nil }
  var y struct{ Projects []map[string]any `yaml:"projects" json:"projects"` }
  _ = yaml.Unmarshal([]byte(raw), &y)
  if len(y.Projects) > 0 { return y.Projects }
  // fallback JSON array plain
  var arr []map[string]any
  _ = json.Unmarshal([]byte(raw), &arr)
  return arr
}

func upsertOpaqueEnvSecret(k8s *kubernetes.Clientset, ns, name string, kv map[string]string, merge bool) (created, updated bool, err error) {
  s, err := k8s.CoreV1().Secrets(ns).Get(context.TODO(), name, metav1.GetOptions{})
  if apierrors.IsNotFound(err) {
    sec := &v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}, Type: v1.SecretTypeOpaque, StringData: map[string]string{}}
    for k, v := range kv { if v != "" { sec.StringData[k] = v } }
    _, err = k8s.CoreV1().Secrets(ns).Create(context.TODO(), sec, metav1.CreateOptions{})
    return true, false, err
  }
  if err != nil { return false, false, err }
  if !merge { // replace
    s.StringData = map[string]string{}
  }
  if s.StringData == nil { s.StringData = map[string]string{} }
  for k, v := range kv { if v != "" { s.StringData[k] = v } }
  _, err = k8s.CoreV1().Secrets(ns).Update(context.TODO(), s, metav1.UpdateOptions{})
  return false, true, err
}
```
