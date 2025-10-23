#!/bin/bash

# scripts/setup-test-repo.sh
# Script to set up a test Git repository with sample Helm charts

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Configuration
REPO_NAME="kargo-test-repo"
REPO_DIR="${PWD}/${REPO_NAME}"
GIT_SERVER="git-server"
GIT_SERVER_PORT="30888"
GIT_SERVER_NAMESPACE="default"

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if git is installed
check_git() {
    if ! command -v git &> /dev/null; then
        log_error "git is not installed. Please install git first."
        exit 1
    fi
    log_info "git is installed"
}

# Check if kubectl is installed and configured
check_kubectl() {
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed. Please install kubectl first."
        exit 1
    fi
    
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster. Please check your kubeconfig."
        exit 1
    fi
    
    log_info "kubectl is installed and configured"
}

# Create test repository directory
create_repo_directory() {
    log_info "Creating test repository directory"
    
    if [ -d "${REPO_DIR}" ]; then
        log_warn "Repository directory already exists: ${REPO_DIR}"
        read -p "Do you want to delete and recreate it? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            rm -rf "${REPO_DIR}"
            log_info "Deleted existing repository directory"
        else
            log_info "Using existing repository directory"
            return 0
        fi
    fi
    
    mkdir -p "${REPO_DIR}"
    log_info "Repository directory created: ${REPO_DIR}"
}

# Initialize Git repository
init_git_repo() {
    log_info "Initializing Git repository"
    
    cd "${REPO_DIR}"
    git init
    git config user.email "test@example.com"
    git config user.name "Test User"
    
    log_info "Git repository initialized"
}

# Create sample Helm charts
create_helm_charts() {
    log_info "Creating sample Helm charts"
    
    # Create web-app chart
    create_web_app_chart
    
    # Create api-service chart
    create_api_service_chart
    
    # Create database chart
    create_database_chart
    
    # Create monitoring chart
    create_monitoring_chart
    
    log_info "Sample Helm charts created"
}

# Create web-app Helm chart
create_web_app_chart() {
    log_info "Creating web-app Helm chart"
    
    # Create chart directory structure
    mkdir -p charts/web-app/templates
    mkdir -p charts/web-app/values
    
    # Create Chart.yaml
    cat > charts/web-app/Chart.yaml << EOF
apiVersion: v2
name: web-app
description: A sample web application Helm chart
type: application
version: 0.1.0
appVersion: "1.0.0"
keywords:
  - web
  - application
  - frontend
home: https://github.com/example/web-app
sources:
  - https://github.com/example/web-app
maintainers:
  - name: Test Maintainer
    email: test@example.com
EOF

    # Create default values.yaml
    cat > charts/web-app/values.yaml << EOF
replicaCount: 1

image:
  repository: nginx
  pullPolicy: IfNotPresent
  tag: "1.20"

nameOverride: ""
fullnameOverride: ""

serviceAccount:
  create: true
  annotations: {}
  name: ""

podAnnotations: {}

podSecurityContext:
  fsGroup: 2000

securityContext:
  capabilities:
    drop:
    - ALL
  readOnlyRootFilesystem: true
  runAsNonRoot: true
  runAsUser: 1000

service:
  type: ClusterIP
  port: 80
  targetPort: 80

ingress:
  enabled: true
  className: "nginx"
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
  hosts:
    - host: web-app.local
      paths:
        - path: /
          pathType: Prefix
  tls: []

resources:
  limits:
    cpu: 100m
    memory: 128Mi
  requests:
    cpu: 50m
    memory: 64Mi

autoscaling:
  enabled: false
  minReplicas: 1
  maxReplicas: 100
  targetCPUUtilizationPercentage: 80

nodeSelector: {}

tolerations: []

affinity: {}
EOF

    # Create deployment template
    cat > charts/web-app/templates/deployment.yaml << EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "web-app.fullname" . }}
  labels:
    {{- include "web-app.labels" . | nindent 4 }}
spec:
  {{- if not .Values.autoscaling.enabled }}
  replicas: {{ .Values.replicaCount }}
  {{- end }}
  selector:
    matchLabels:
      {{- include "web-app.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      annotations:
        checksum/config: {{ include (print $.Template.BasePath "/configmap.yaml") . | sha256sum }}
        {{- with .Values.podAnnotations }}
        {{- toYaml . | nindent 8 }}
        {{- end }}
      labels:
        {{- include "web-app.selectorLabels" . | nindent 8 }}
    spec:
      {{- with .Values.imagePullSecrets }}
      imagePullSecrets:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      serviceAccountName: {{ include "web-app.serviceAccountName" . }}
      securityContext:
        {{- toYaml .Values.podSecurityContext | nindent 8 }}
      containers:
        - name: {{ .Chart.Name }}
          securityContext:
            {{- toYaml .Values.securityContext | nindent 12 }}
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}"
          imagePullPolicy: {{ .Values.image.pullPolicy }}
          ports:
            - name: http
              containerPort: 80
              protocol: TCP
          livenessProbe:
            httpGet:
              path: /
              port: http
          readinessProbe:
            httpGet:
              path: /
              port: http
          resources:
            {{- toYaml .Values.resources | nindent 12 }}
      {{- with .Values.nodeSelector }}
      nodeSelector:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.affinity }}
      affinity:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.tolerations }}
      tolerations:
        {{- toYaml . | nindent 8 }}
      {{- end }}
EOF

    # Create service template
    cat > charts/web-app/templates/service.yaml << EOF
apiVersion: v1
kind: Service
metadata:
  name: {{ include "web-app.fullname" . }}
  labels:
    {{- include "web-app.labels" . | nindent 4 }}
spec:
  type: {{ .Values.service.type }}
  ports:
    - port: {{ .Values.service.port }}
      targetPort: {{ .Values.service.targetPort }}
      protocol: TCP
      name: http
  selector:
    {{- include "web-app.selectorLabels" . | nindent 4 }}
EOF

    # Create ingress template
    cat > charts/web-app/templates/ingress.yaml << EOF
{{- if .Values.ingress.enabled -}}
{{- \$fullName := include "web-app.fullname" . -}}
{{- \$svcPort := .Values.service.port -}}
{{- if and .Values.ingress.className (not (semverCompare ">=1.18-0" .Capabilities.KubeVersion.GitVersion)) }}
  {{- if not (hasKey .Values.ingress.annotations "kubernetes.io/ingress.class") }}
  {{- \$ := set .Values.ingress.annotations "kubernetes.io/ingress.class" .Values.ingress.className}}
  {{- end }}
{{- end }}
{{- if semverCompare ">=1.19-0" .Capabilities.KubeVersion.GitVersion -}}
apiVersion: networking.k8s.io/v1
{{- else if semverCompare ">=1.14-0" .Capabilities.KubeVersion.GitVersion -}}
apiVersion: networking.k8s.io/v1beta1
{{- else -}}
apiVersion: extensions/v1beta1
{{- end }}
kind: Ingress
metadata:
  name: {{ \$fullName }}
  labels:
    {{- include "web-app.labels" . | nindent 4 }}
  {{- with .Values.ingress.annotations }}
  annotations:
    {{- toYaml . | nindent 4 }}
  {{- end }}
spec:
  {{- if and .Values.ingress.className (semverCompare ">=1.18-0" .Capabilities.KubeVersion.GitVersion) }}
  ingressClassName: {{ .Values.ingress.className }}
  {{- end }}
  {{- if .Values.ingress.tls }}
  tls:
    {{- range .Values.ingress.tls }}
    - hosts:
        {{- range .hosts }}
        - {{ . | quote }}
        {{- end }}
      secretName: {{ .secretName }}
    {{- end }}
  {{- end }}
  rules:
    {{- range .Values.ingress.hosts }}
    - host: {{ .host | quote }}
      http:
        paths:
          {{- range .paths }}
          - path: {{ .path }}
            {{- if and .pathType (semverCompare ">=1.18-0" \$.Capabilities.KubeVersion.GitVersion) }}
            pathType: {{ .pathType }}
            {{- end }}
            backend:
              {{- if semverCompare ">=1.19-0" \$.Capabilities.KubeVersion.GitVersion }}
              service:
                name: {{ \$fullName }}
                port:
                  number: {{ \$svcPort }}
              {{- else }}
              serviceName: {{ \$fullName }}
              servicePort: {{ \$svcPort }}
              {{- end }}
          {{- end }}
    {{- end }}
{{- end }}
EOF

    # Create helpers template
    cat > charts/web-app/templates/_helpers.tpl << EOF
{{/*
Expand the name of the chart.
*/}}
{{- define "web-app.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "web-app.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- \$name := default .Chart.Name .Values.nameOverride }}
{{- if contains \$name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name \$name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "web-app.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "web-app.labels" -}}
helm.sh/chart: {{ include "web-app.chart" . }}
{{ include "web-app.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "web-app.selectorLabels" -}}
app.kubernetes.io/name: {{ include "web-app.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "web-app.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "web-app.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}
EOF

    # Create environment-specific values
    mkdir -p charts/web-app/values
    
    # Development values
    cat > charts/web-app/values/dev.yaml << EOF
replicaCount: 1

image:
  tag: "dev"

service:
  type: NodePort
  nodePort: 30081

ingress:
  enabled: true
  hosts:
    - host: web-app-dev.local
      paths:
        - path: /
          pathType: Prefix

resources:
  limits:
    cpu: 200m
    memory: 256Mi
  requests:
    cpu: 100m
    memory: 128Mi
EOF

    # Staging values
    cat > charts/web-app/values/staging.yaml << EOF
replicaCount: 2

image:
  tag: "staging"

ingress:
  enabled: true
  hosts:
    - host: web-app-staging.local
      paths:
        - path: /
          pathType: Prefix

resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 250m
    memory: 256Mi

autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 5
EOF

    # Production values
    cat > charts/web-app/values/prod.yaml << EOF
replicaCount: 3

image:
  tag: "latest"

ingress:
  enabled: true
  hosts:
    - host: web-app-prod.local
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: web-app-tls
      hosts:
        - web-app-prod.local

resources:
  limits:
    cpu: 1000m
    memory: 1Gi
  requests:
    cpu: 500m
    memory: 512Mi

autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 10
EOF
}

# Create api-service Helm chart
create_api_service_chart() {
    log_info "Creating api-service Helm chart"
    
    # Create chart directory structure
    mkdir -p charts/api-service/templates
    mkdir -p charts/api-service/values
    
    # Create Chart.yaml
    cat > charts/api-service/Chart.yaml << EOF
apiVersion: v2
name: api-service
description: A sample API service Helm chart
type: application
version: 0.1.0
appVersion: "1.0.0"
keywords:
  - api
  - service
  - backend
home: https://github.com/example/api-service
sources:
  - https://github.com/example/api-service
maintainers:
  - name: Test Maintainer
    email: test@example.com
EOF

    # Create default values.yaml
    cat > charts/api-service/values.yaml << EOF
replicaCount: 1

image:
  repository: nginx
  pullPolicy: IfNotPresent
  tag: "1.20"

nameOverride: ""
fullnameOverride: ""

serviceAccount:
  create: true
  annotations: {}
  name: ""

podAnnotations: {}

podSecurityContext:
  fsGroup: 2000

securityContext:
  capabilities:
    drop:
    - ALL
  readOnlyRootFilesystem: true
  runAsNonRoot: true
  runAsUser: 1000

service:
  type: ClusterIP
  port: 8080
  targetPort: 8080

ingress:
  enabled: true
  className: "nginx"
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
  hosts:
    - host: api-service.local
      paths:
        - path: /
          pathType: Prefix
  tls: []

resources:
  limits:
    cpu: 200m
    memory: 256Mi
  requests:
    cpu: 100m
    memory: 128Mi

autoscaling:
  enabled: false
  minReplicas: 1
  maxReplicas: 100
  targetCPUUtilizationPercentage: 80

nodeSelector: {}

tolerations: []

affinity: {}
EOF

    # Create deployment template
    cat > charts/api-service/templates/deployment.yaml << EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "api-service.fullname" . }}
  labels:
    {{- include "api-service.labels" . | nindent 4 }}
spec:
  {{- if not .Values.autoscaling.enabled }}
  replicas: {{ .Values.replicaCount }}
  {{- end }}
  selector:
    matchLabels:
      {{- include "api-service.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      annotations:
        checksum/config: {{ include (print $.Template.BasePath "/configmap.yaml") . | sha256sum }}
        {{- with .Values.podAnnotations }}
        {{- toYaml . | nindent 8 }}
        {{- end }}
      labels:
        {{- include "api-service.selectorLabels" . | nindent 8 }}
    spec:
      {{- with .Values.imagePullSecrets }}
      imagePullSecrets:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      serviceAccountName: {{ include "api-service.serviceAccountName" . }}
      securityContext:
        {{- toYaml .Values.podSecurityContext | nindent 8 }}
      containers:
        - name: {{ .Chart.Name }}
          securityContext:
            {{- toYaml .Values.securityContext | nindent 12 }}
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}"
          imagePullPolicy: {{ .Values.image.pullPolicy }}
          ports:
            - name: http
              containerPort: 8080
              protocol: TCP
          livenessProbe:
            httpGet:
              path: /health
              port: http
          readinessProbe:
            httpGet:
              path: /ready
              port: http
          resources:
            {{- toYaml .Values.resources | nindent 12 }}
      {{- with .Values.nodeSelector }}
      nodeSelector:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.affinity }}
      affinity:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.tolerations }}
      tolerations:
        {{- toYaml . | nindent 8 }}
      {{- end }}
EOF

    # Create service template
    cat > charts/api-service/templates/service.yaml << EOF
apiVersion: v1
kind: Service
metadata:
  name: {{ include "api-service.fullname" . }}
  labels:
    {{- include "api-service.labels" . | nindent 4 }}
spec:
  type: {{ .Values.service.type }}
  ports:
    - port: {{ .Values.service.port }}
      targetPort: {{ .Values.service.targetPort }}
      protocol: TCP
      name: http
  selector:
    {{- include "api-service.selectorLabels" . | nindent 4 }}
EOF

    # Create helpers template
    cat > charts/api-service/templates/_helpers.tpl << EOF
{{/*
Expand the name of the chart.
*/}}
{{- define "api-service.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "api-service.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- \$name := default .Chart.Name .Values.nameOverride }}
{{- if contains \$name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name \$name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "api-service.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "api-service.labels" -}}
helm.sh/chart: {{ include "api-service.chart" . }}
{{ include "api-service.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "api-service.selectorLabels" -}}
app.kubernetes.io/name: {{ include "api-service.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "api-service.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "api-service.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}
EOF

    # Create environment-specific values
    mkdir -p charts/api-service/values
    
    # Development values
    cat > charts/api-service/values/dev.yaml << EOF
replicaCount: 1

image:
  tag: "dev"

service:
  type: NodePort
  nodePort: 30082

ingress:
  enabled: true
  hosts:
    - host: api-service-dev.local
      paths:
        - path: /
          pathType: Prefix

resources:
  limits:
    cpu: 300m
    memory: 384Mi
  requests:
    cpu: 150m
    memory: 192Mi
EOF

    # Staging values
    cat > charts/api-service/values/staging.yaml << EOF
replicaCount: 2

image:
  tag: "staging"

ingress:
  enabled: true
  hosts:
    - host: api-service-staging.local
      paths:
        - path: /
          pathType: Prefix

resources:
  limits:
    cpu: 600m
    memory: 768Mi
  requests:
    cpu: 300m
    memory: 384Mi

autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 5
EOF

    # Production values
    cat > charts/api-service/values/prod.yaml << EOF
replicaCount: 3

image:
  tag: "latest"

ingress:
  enabled: true
  hosts:
    - host: api-service-prod.local
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: api-service-tls
      hosts:
        - api-service-prod.local

resources:
  limits:
    cpu: 1200m
    memory: 1.5Gi
  requests:
    cpu: 600m
    memory: 768Mi

autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 10
EOF
}

# Create database Helm chart
create_database_chart() {
    log_info "Creating database Helm chart"
    
    # Create chart directory structure
    mkdir -p charts/database/templates
    mkdir -p charts/database/values
    
    # Create Chart.yaml
    cat > charts/database/Chart.yaml << EOF
apiVersion: v2
name: database
description: A sample database Helm chart
type: application
version: 0.1.0
appVersion: "1.0.0"
keywords:
  - database
  - postgresql
  - data
home: https://github.com/example/database
sources:
  - https://github.com/example/database
maintainers:
  - name: Test Maintainer
    email: test@example.com
EOF

    # Create default values.yaml
    cat > charts/database/values.yaml << EOF
replicaCount: 1

image:
  repository: postgres
  pullPolicy: IfNotPresent
  tag: "13"

nameOverride: ""
fullnameOverride: ""

auth:
  postgresPassword: "postgres"
  database: "appdb"

service:
  type: ClusterIP
  port: 5432

persistence:
  enabled: true
  storageClass: "standard"
  size: 1Gi

resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 250m
    memory: 256Mi

nodeSelector: {}

tolerations: []

affinity: {}
EOF

    # Create deployment template
    cat > charts/database/templates/deployment.yaml << EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "database.fullname" . }}
  labels:
    {{- include "database.labels" . | nindent 4 }}
spec:
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels:
      {{- include "database.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      annotations:
        {{- with .Values.podAnnotations }}
        {{- toYaml . | nindent 8 }}
        {{- end }}
      labels:
        {{- include "database.selectorLabels" . | nindent 8 }}
    spec:
      {{- with .Values.imagePullSecrets }}
      imagePullSecrets:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      securityContext:
        {{- toYaml .Values.podSecurityContext | nindent 8 }}
      containers:
        - name: {{ .Chart.Name }}
          securityContext:
            {{- toYaml .Values.securityContext | nindent 12 }}
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}"
          imagePullPolicy: {{ .Values.image.pullPolicy }}
          ports:
            - name: postgres
              containerPort: 5432
              protocol: TCP
          env:
            - name: POSTGRES_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: {{ include "database.fullname" . }}
                  key: postgres-password
            - name: POSTGRES_DB
              value: {{ .Values.auth.database }}
          volumeMounts:
            - name: data
              mountPath: /var/lib/postgresql/data
          resources:
            {{- toYaml .Values.resources | nindent 12 }}
      {{- with .Values.nodeSelector }}
      nodeSelector:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.affinity }}
      affinity:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.tolerations }}
      tolerations:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      volumes:
        - name: data
          {{- if .Values.persistence.enabled }}
          persistentVolumeClaim:
            claimName: {{ include "database.fullname" . }}
          {{- else }}
          emptyDir: {}
          {{- end }}
EOF

    # Create service template
    cat > charts/database/templates/service.yaml << EOF
apiVersion: v1
kind: Service
metadata:
  name: {{ include "database.fullname" . }}
  labels:
    {{- include "database.labels" . | nindent 4 }}
spec:
  type: {{ .Values.service.type }}
  ports:
    - port: {{ .Values.service.port }}
      targetPort: postgres
      protocol: TCP
      name: postgres
  selector:
    {{- include "database.selectorLabels" . | nindent 4 }}
EOF

    # Create PVC template
    cat > charts/database/templates/pvc.yaml << EOF
{{- if .Values.persistence.enabled }}
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: {{ include "database.fullname" . }}
  labels:
    {{- include "database.labels" . | nindent 4 }}
spec:
  accessModes:
    - ReadWriteOnce
  storageClassName: {{ .Values.persistence.storageClass }}
  resources:
    requests:
      storage: {{ .Values.persistence.size }}
{{- end }}
EOF

    # Create secret template
    cat > charts/database/templates/secret.yaml << EOF
apiVersion: v1
kind: Secret
metadata:
  name: {{ include "database.fullname" . }}
  labels:
    {{- include "database.labels" . | nindent 4 }}
type: Opaque
data:
  postgres-password: {{ .Values.auth.postgresPassword | b64enc }}
EOF

    # Create helpers template
    cat > charts/database/templates/_helpers.tpl << EOF
{{/*
Expand the name of the chart.
*/}}
{{- define "database.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "database.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- \$name := default .Chart.Name .Values.nameOverride }}
{{- if contains \$name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name \$name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "database.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "database.labels" -}}
helm.sh/chart: {{ include "database.chart" . }}
{{ include "database.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "database.selectorLabels" -}}
app.kubernetes.io/name: {{ include "database.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
EOF

    # Create environment-specific values
    mkdir -p charts/database/values
    
    # Development values
    cat > charts/database/values/dev.yaml << EOF
replicaCount: 1

persistence:
  enabled: true
  size: 1Gi

resources:
  limits:
    cpu: 300m
    memory: 384Mi
  requests:
    cpu: 150m
    memory: 192Mi
EOF

    # Staging values
    cat > charts/database/values/staging.yaml << EOF
replicaCount: 1

persistence:
  enabled: true
  size: 5Gi

resources:
  limits:
    cpu: 600m
    memory: 768Mi
  requests:
    cpu: 300m
    memory: 384Mi
EOF

    # Production values
    cat > charts/database/values/prod.yaml << EOF
replicaCount: 1

persistence:
  enabled: true
  size: 20Gi

resources:
  limits:
    cpu: 1200m
    memory: 1.5Gi
  requests:
    cpu: 600m
    memory: 768Mi
EOF
}

# Create monitoring Helm chart
create_monitoring_chart() {
    log_info "Creating monitoring Helm chart"
    
    # Create chart directory structure
    mkdir -p charts/monitoring/templates
    mkdir -p charts/monitoring/values
    
    # Create Chart.yaml
    cat > charts/monitoring/Chart.yaml << EOF
apiVersion: v2
name: monitoring
description: A sample monitoring Helm chart
type: application
version: 0.1.0
appVersion: "1.0.0"
keywords:
  - monitoring
  - prometheus
  - grafana
home: https://github.com/example/monitoring
sources:
  - https://github.com/example/monitoring
maintainers:
  - name: Test Maintainer
    email: test@example.com
EOF

    # Create default values.yaml
    cat > charts/monitoring/values.yaml << EOF
replicaCount: 1

prometheus:
  image:
    repository: prom/prometheus
    tag: "latest"
  port: 9090
  resources:
    limits:
      cpu: 500m
      memory: 512Mi
    requests:
      cpu: 250m
      memory: 256Mi

grafana:
  image:
    repository: grafana/grafana
    tag: "latest"
  port: 3000
  resources:
    limits:
      cpu: 200m
   
      memory: 256Mi
    requests:
      cpu: 100m
      memory: 128Mi

service:
  type: ClusterIP

ingress:
  enabled: true
  className: "nginx"
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
  hosts:
    - host: monitoring.local
      paths:
        - path: /
          pathType: Prefix
  tls: []

nodeSelector: {}

tolerations: []

affinity: {}
EOF

    # Create Prometheus deployment template
    cat > charts/monitoring/templates/prometheus-deployment.yaml << EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "monitoring.fullname" . }}-prometheus
  labels:
    {{- include "monitoring.labels" . | nindent 4 }}
    app: prometheus
spec:
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels:
      {{- include "monitoring.selectorLabels" . | nindent 6 }}
      app: prometheus
  template:
    metadata:
      annotations:
        {{- with .Values.podAnnotations }}
        {{- toYaml . | nindent 8 }}
        {{- end }}
      labels:
        {{- include "monitoring.selectorLabels" . | nindent 8 }}
        app: prometheus
    spec:
      securityContext:
        {{- toYaml .Values.podSecurityContext | nindent 8 }}
      containers:
        - name: prometheus
          securityContext:
            {{- toYaml .Values.securityContext | nindent 12 }}
          image: "{{ .Values.prometheus.image.repository }}:{{ .Values.prometheus.image.tag }}"
          imagePullPolicy: IfNotPresent
          ports:
            - name: http
              containerPort: {{ .Values.prometheus.port }}
              protocol: TCP
          resources:
            {{- toYaml .Values.prometheus.resources | nindent 12 }}
      {{- with .Values.nodeSelector }}
      nodeSelector:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.affinity }}
      affinity:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.tolerations }}
      tolerations:
        {{- toYaml . | nindent 8 }}
      {{- end }}
EOF

    # Create Prometheus service template
    cat > charts/monitoring/templates/prometheus-service.yaml << EOF
apiVersion: v1
kind: Service
metadata:
  name: {{ include "monitoring.fullname" . }}-prometheus
  labels:
    {{- include "monitoring.labels" . | nindent 4 }}
    app: prometheus
spec:
  type: {{ .Values.service.type }}
  ports:
    - port: {{ .Values.prometheus.port }}
      targetPort: http
      protocol: TCP
      name: http
  selector:
    {{- include "monitoring.selectorLabels" . | nindent 4 }}
    app: prometheus
EOF

    # Create Grafana deployment template
    cat > charts/monitoring/templates/grafana-deployment.yaml << EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "monitoring.fullname" . }}-grafana
  labels:
    {{- include "monitoring.labels" . | nindent 4 }}
    app: grafana
spec:
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels:
      {{- include "monitoring.selectorLabels" . | nindent 6 }}
      app: grafana
  template:
    metadata:
      annotations:
        {{- with .Values.podAnnotations }}
        {{- toYaml . | nindent 8 }}
        {{- end }}
      labels:
        {{- include "monitoring.selectorLabels" . | nindent 8 }}
        app: grafana
    spec:
      securityContext:
        {{- toYaml .Values.podSecurityContext | nindent 8 }}
      containers:
        - name: grafana
          securityContext:
            {{- toYaml .Values.securityContext | nindent 12 }}
          image: "{{ .Values.grafana.image.repository }}:{{ .Values.grafana.image.tag }}"
          imagePullPolicy: IfNotPresent
          ports:
            - name: http
              containerPort: {{ .Values.grafana.port }}
              protocol: TCP
          resources:
            {{- toYaml .Values.grafana.resources | nindent 12 }}
      {{- with .Values.nodeSelector }}
      nodeSelector:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.affinity }}
      affinity:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.tolerations }}
      tolerations:
        {{- toYaml . | nindent 8 }}
      {{- end }}
EOF

    # Create Grafana service template
    cat > charts/monitoring/templates/grafana-service.yaml << EOF
apiVersion: v1
kind: Service
metadata:
  name: {{ include "monitoring.fullname" . }}-grafana
  labels:
    {{- include "monitoring.labels" . | nindent 4 }}
    app: grafana
spec:
  type: {{ .Values.service.type }}
  ports:
    - port: {{ .Values.grafana.port }}
      targetPort: http
      protocol: TCP
      name: http
  selector:
    {{- include "monitoring.selectorLabels" . | nindent 4 }}
    app: grafana
EOF

    # Create helpers template
    cat > charts/monitoring/templates/_helpers.tpl << EOF
{{/*
Expand the name of the chart.
*/}}
{{- define "monitoring.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "monitoring.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- \$name := default .Chart.Name .Values.nameOverride }}
{{- if contains \$name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name \$name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "monitoring.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "monitoring.labels" -}}
helm.sh/chart: {{ include "monitoring.chart" . }}
{{ include "monitoring.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "monitoring.selectorLabels" -}}
app.kubernetes.io/name: {{ include "monitoring.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
EOF

    # Create environment-specific values
    mkdir -p charts/monitoring/values
    
    # Development values
    cat > charts/monitoring/values/dev.yaml << EOF
replicaCount: 1

prometheus:
  resources:
    limits:
      cpu: 300m
      memory: 384Mi
    requests:
      cpu: 150m
      memory: 192Mi

grafana:
  resources:
    limits:
      cpu: 150m
      memory: 192Mi
    requests:
      cpu: 75m
      memory: 96Mi
EOF

    # Staging values
    cat > charts/monitoring/values/staging.yaml << EOF
replicaCount: 1

prometheus:
  resources:
    limits:
      cpu: 600m
      memory: 768Mi
    requests:
      cpu: 300m
      memory: 384Mi

grafana:
  resources:
    limits:
      cpu: 300m
      memory: 384Mi
    requests:
      cpu: 150m
      memory: 192Mi
EOF

    # Production values
    cat > charts/monitoring/values/prod.yaml << EOF
replicaCount: 2

prometheus:
  resources:
    limits:
      cpu: 1200m
      memory: 1.5Gi
    requests:
      cpu: 600m
      memory: 768Mi

grafana:
  resources:
    limits:
      cpu: 600m
      memory: 768Mi
    requests:
      cpu: 300m
      memory: 384Mi
EOF
}

# Create README for the repository
create_readme() {
    log_info "Creating README for the repository"
    
    cat > README.md << EOF
# Kargo Test Repository

This repository contains sample Helm charts for testing the kargo-bootstrap tool.

## Repository Structure

\`\`\`
.
├── charts/
│   ├── web-app/
│   │   ├── Chart.yaml
│   │   ├── values.yaml
│   │   ├── values/
│   │   │   ├── dev.yaml
│   │   │   ├── staging.yaml
│   │   │   └── prod.yaml
│   │   └── templates/
│   │       ├── deployment.yaml
│   │       ├── service.yaml
│   │       ├── ingress.yaml
│   │       └── _helpers.tpl
│   ├── api-service/
│   │   ├── Chart.yaml
│   │   ├── values.yaml
│   │   ├── values/
│   │   │   ├── dev.yaml
│   │   │   ├── staging.yaml
│   │   │   └── prod.yaml
│   │   └── templates/
│   │       ├── deployment.yaml
│   │       ├── service.yaml
│   │       └── _helpers.tpl
│   ├── database/
│   │   ├── Chart.yaml
│   │   ├── values.yaml
│   │   ├── values/
│   │   │   ├── dev.yaml
│   │   │   ├── staging.yaml
│   │   │   └── prod.yaml
│   │   └── templates/
│   │       ├── deployment.yaml
│   │       ├── service.yaml
│   │       ├── pvc.yaml
│   │       ├── secret.yaml
│   │       └── _helpers.tpl
│   └── monitoring/
│       ├── Chart.yaml
│       ├── values.yaml
│       ├── values/
│       │   ├── dev.yaml
│       │   ├── staging.yaml
│       │   └── prod.yaml
│       └── templates/
│           ├── prometheus-deployment.yaml
│           ├── prometheus-service.yaml
│           ├── grafana-deployment.yaml
│           ├── grafana-service.yaml
│           └── _helpers.tpl
└── README.md
\`\`\`

## Charts

### web-app
A sample web application using Nginx.
- **Version**: 0.1.0
- **App Version**: 1.0.0
- **Environments**: dev, staging, prod

### api-service
A sample API service using Nginx.
- **Version**: 0.1.0
- **App Version**: 1.0.0
- **Environments**: dev, staging, prod

### database
A sample PostgreSQL database.
- **Version**: 0.1.0
- **App Version**: 1.0.0
- **Environments**: dev, staging, prod

### monitoring
A sample monitoring stack with Prometheus and Grafana.
- **Version**: 0.1.0
- **App Version**: 1.0.0
- **Environments**: dev, staging, prod

## Usage

This repository is designed to be used with the kargo-bootstrap tool for testing deployment workflows.

### Testing with kargo-bootstrap

1. Clone this repository:
   \`\`\`bash
   git clone <repository-url>
   cd kargo-test-repo
   \`\`\`

2. Use kargo-bootstrap to deploy charts:
   \`\`\`bash
   kargo-bootstrap deploy --repo <repository-url> --chart web-app --env dev
   \`\`\`

### Testing Different Environments

Each chart has environment-specific values files:
- \`values/dev.yaml\` - Development environment
- \`values/staging.yaml\` - Staging environment
- \`values/prod.yaml\` - Production environment

## Git Repository Configuration

This repository is configured to be accessible from within the Kubernetes cluster for Argo CD.

### Repository URL
- HTTP: http://git-server.default.svc.cluster.local:30888/kargo-test-repo.git
- SSH: git@git-server.default.svc.cluster.local:kargo-test-repo.git

### Access from Argo CD

Add this repository to Argo CD using the HTTP URL for cluster access:
- URL: http://git-server.default.svc.cluster.local:30888/kargo-test-repo.git
- Username: git
- Password: (no password required for testing)

## Chart Paths

This repository contains charts in the following paths:
- \`charts/web-app\`
- \`charts/api-service\`
- \`charts/database\`
- \`charts/monitoring\`

Each chart can be deployed to different environments using the appropriate values file.
EOF

    log_info "README created"
}

# Commit all changes to Git
commit_changes() {
    log_info "Committing changes to Git"
    
    cd "${REPO_DIR}"
    
    # Add all files
    git add .
    
    # Commit
    git commit -m "Initial commit with sample Helm charts"
    
    log_info "Changes committed to Git"
}

# Set up Git server in the cluster
setup_git_server() {
    log_info "Setting up Git server in the cluster"
    
    # Create Git server deployment
    cat > git-server-deployment.yaml << EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ${GIT_SERVER}
  namespace: ${GIT_SERVER_NAMESPACE}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ${GIT_SERVER}
  template:
    metadata:
      labels:
        app: ${GIT_SERVER}
    spec:
      containers:
      - name: ${GIT_SERVER}
        image: gitea/gitea:latest
        ports:
        - containerPort: 22
        - containerPort: 3000
        env:
        - name: ROOT_URL
          value: "http://localhost:3000"
        - name: SSH_DOMAIN
          value: "git-server.default.svc.cluster.local"
        - name: SSH_PORT
          value: "22"
        - name: DISABLE_REGISTRATION
          value: "true"
        volumeMounts:
        - name: git-data
          mountPath: /data
      volumes:
      - name: git-data
        emptyDir: {}
EOF

    # Create Git server service
    cat > git-server-service.yaml << EOF
apiVersion: v1
kind: Service
metadata:
  name: ${GIT_SERVER}
  namespace: ${GIT_SERVER_NAMESPACE}
spec:
  type: NodePort
  ports:
  - name: ssh
    port: 22
    targetPort: 22
    nodePort: ${GIT_SERVER_PORT}
  - name: http
    port: 3000
    targetPort: 3000
    nodePort: 30889
  selector:
    app: ${GIT_SERVER}
EOF

    # Apply Git server configurations
    kubectl apply -f git-server-deployment.yaml
    kubectl apply -f git-server-service.yaml
    
    # Wait for Git server to be ready
    kubectl wait --for=condition=available --timeout=300s deployment/${GIT_SERVER} -n ${GIT_SERVER_NAMESPACE}
    
    log_info "Git server is ready"
    log_info "SSH URL: ssh://git@git-server.default.svc.cluster.local:${GIT_SERVER_PORT}/kargo-test-repo.git"
    log_info "HTTP URL: http://git-server.default.svc.cluster.local:30889/kargo-test-repo.git"
}

# Push repository to Git server
push_to_git_server() {
    log_info "Pushing repository to Git server"
    
    cd "${REPO_DIR}"
    
    # Add Git server as remote
    git remote add origin ssh://git@git-server.default.svc.cluster.local:${GIT_SERVER_PORT}/kargo-test-repo.git
    
    # Push to Git server
    git push -u origin master
    
    log_info "Repository pushed to Git server"
}

# Cleanup function
cleanup() {
    log_info "Cleaning up temporary files"
    rm -f git-server-deployment.yaml
    rm -f git-server-service.yaml
}

# Print repository information
print_repo_info() {
    log_info "Test repository setup completed!"
    echo ""
    echo "Repository Information:"
    echo "====================="
    echo "Repository directory: ${REPO_DIR}"
    echo "Repository name: ${REPO_NAME}"
    echo ""
    echo "Git Server Information:"
    echo "======================"
    echo "SSH URL: ssh://git@git-server.default.svc.cluster.local:${GIT_SERVER_PORT}/${REPO_NAME}.git"
    echo "HTTP URL: http://git-server.default.svc.cluster.local:30889/${REPO_NAME}.git"
    echo ""
    echo "Chart Paths:"
    echo "============"
    echo "- charts/web-app"
    echo "- charts/api-service"
    echo "- charts/database"
    echo "- charts/monitoring"
    echo ""
    echo "Environments:"
    echo "============="
    echo "- dev"
    echo "- staging"
    echo "- prod"
    echo ""
    echo "Usage with kargo-bootstrap:"
    echo "=========================="
    echo "kargo-bootstrap deploy --repo ssh://git@git-server.default.svc.cluster.local:${GIT_SERVER_PORT}/${REPO_NAME}.git --chart web-app --env dev"
}

# Main execution
main() {
    log_info "Setting up test Git repository for kargo-bootstrap testing"
    
    # Set up cleanup trap
    trap cleanup EXIT
    
    # Run checks
    check_git
    check_kubectl
    
    # Create repository directory
    create_repo_directory
    
    # Initialize Git repository
    init_git_repo
    
    # Create Helm charts
    create_helm_charts
    
    # Create README
    create_readme
    
    # Commit changes
    commit_changes
    
    # Set up Git server in cluster
    setup_git_server
    
    # Push repository to Git server
    push_to_git_server
    
    # Print repository information
    print_repo_info
    
    log_info "Test repository setup completed successfully!"
}

# Run main function
main "$@"