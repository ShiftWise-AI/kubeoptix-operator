local json = require("dkjson")

local cjson = {
  encode = function(value)
    local encoded, err = json.encode(value)
    if not encoded then error(err) end
    return encoded
  end,
  decode = function(value)
    local decoded, _, err = json.decode(value)
    return decoded, err
  end,
}

local poll_seconds = tonumber(os.getenv("RECONCILE_INTERVAL_SECONDS")) or 30
local kubectl = os.getenv("KUBECTL_BIN") or "kubectl"

local default_images = {
  harvester = os.getenv("KUBEOPTIX_HARVESTER_IMAGE") or "quay.io/shiftwise-ai/kubeoptix-harvester:latest",
  analyzer = os.getenv("KUBEOPTIX_ANALYZER_IMAGE") or "quay.io/shiftwise-ai/kubeoptix-analyzer:latest",
  coreAi = os.getenv("KUBEOPTIX_CORE_AI_IMAGE") or "quay.io/shiftwise-ai/kubeoptix-core-ai:latest",
  configurations = os.getenv("KUBEOPTIX_CONFIGURATIONS_IMAGE") or "quay.io/shiftwise-ai/kubeoptix-configurations:latest",
  reporter = os.getenv("KUBEOPTIX_REPORTER_IMAGE") or "quay.io/shiftwise-ai/kubeoptix-reporter:latest",
  dashboard = os.getenv("KUBEOPTIX_DASHBOARD_IMAGE") or "quay.io/shiftwise-ai/kubeoptix-dashboard:latest",
  postgres = os.getenv("POSTGRES_IMAGE") or "registry.redhat.io/rhel9/postgresql-16:latest",
}

local function log(level, message, fields)
  local entry = fields or {}
  entry.timestamp = os.date("!%Y-%m-%dT%H:%M:%SZ")
  entry.level = level
  entry.message = message
  print(cjson.encode(entry))
end

local function shell_quote(value)
  return "'" .. tostring(value):gsub("'", "'\\''") .. "'"
end

local function run(command, input)
  local input_path
  if input then
    input_path = os.tmpname()
    local file = assert(io.open(input_path, "w"))
    file:write(input)
    file:close()
    command = command .. " < " .. shell_quote(input_path)
  end
  local pipe = io.popen(command .. " 2>&1")
  local output = pipe:read("*a")
  local ok, _, code = pipe:close()
  if input_path then os.remove(input_path) end
  return ok and (code == nil or code == 0), output
end

local function get_json(arguments)
  local ok, output = run(kubectl .. " " .. arguments .. " -o json")
  if not ok then return nil, output end
  local value, err = cjson.decode(output)
  if not value then return nil, err end
  return value
end

local function object(api_version, kind, metadata, spec)
  local value = { apiVersion = api_version, kind = kind, metadata = metadata }
  if spec then value.spec = spec end
  return value
end

local function labels(name, instance)
  return {
    ["app.kubernetes.io/managed-by"] = "shiftwise-lua-operator",
    ["app.kubernetes.io/part-of"] = "kubeoptix",
    ["app.kubernetes.io/name"] = name,
    ["app.kubernetes.io/instance"] = instance,
  }
end

local function enabled(spec, name)
  return not (spec.components and spec.components[name] and spec.components[name].enabled == false)
end

local function env_secret(name, secret_name, key)
  return { name = name, valueFrom = { secretKeyRef = { name = secret_name, key = key } } }
end

local function deployment(component, namespace, instance, image, env, mounts, claim_name)
  local component_labels = labels(component, instance)
  local health_path = "/health"
  if component == "kubeoptix-configurations" then health_path = "/q/health/ready" end
  if component == "kubeoptix-dashboard" then health_path = "/" end
  return object("apps/v1", "Deployment", { name = component, namespace = namespace, labels = component_labels }, {
    replicas = 1,
    selector = { matchLabels = component_labels },
    template = { metadata = { labels = component_labels }, spec = {
      serviceAccountName = component == "kubeoptix-harvester" and "kubeoptix-harvester" or nil,
      containers = {{ name = component, image = image, imagePullPolicy = "IfNotPresent", env = env,
        ports = {{ name = "http", containerPort = 8000 }},
        volumeMounts = mounts,
        readinessProbe = { httpGet = { path = health_path, port = "http" }, initialDelaySeconds = 10, periodSeconds = 10 },
        livenessProbe = { httpGet = { path = health_path, port = "http" }, initialDelaySeconds = 20, periodSeconds = 20 },
        resources = { requests = { cpu = "100m", memory = "128Mi" }, limits = { cpu = "1", memory = "1Gi" } },
      }},
      volumes = mounts and {{ name = "kubeoptix-data", persistentVolumeClaim = { claimName = claim_name }}} or nil,
    }},
  })
end

local function service(component, namespace, instance)
  return object("v1", "Service", { name = component, namespace = namespace, labels = labels(component, instance) }, {
    selector = labels(component, instance), ports = {{ name = "http", port = 8000, targetPort = "http" }},
  })
end

local function build_resources(resource)
  local spec = resource.spec or {}
  local metadata = resource.metadata or {}
  local namespace = spec.targetNamespace or metadata.namespace
  local instance = metadata.name
  local images = setmetatable(spec.images or {}, { __index = default_images })
  local storage = spec.storage or {}
  local credentials = spec.credentials or {}
  local secret_name = credentials.existingSecret or "kubeoptix-credentials"
  local data_claim = storage.existingClaim or "kubeoptix-data"
  local resources = {}

  table.insert(resources, object("v1", "Namespace", { name = namespace, labels = labels("kubeoptix", instance) }))
  if not credentials.existingSecret then
    table.insert(resources, object("v1", "Secret", { name = secret_name, namespace = namespace, labels = labels("credentials", instance) },
      { type = "Opaque", stringData = {
        POSTGRESQL_USER = credentials.postgresUser or "kubeoptix",
        POSTGRESQL_PASSWORD = credentials.postgresPassword or "change-me-before-production",
        POSTGRESQL_DATABASE = credentials.postgresDatabase or "kubeoptix",
        LLM_API_KEY = credentials.llmApiKey or "",
      }}))
  end
  table.insert(resources, object("v1", "ConfigMap", { name = "kubeoptix-endpoints", namespace = namespace, labels = labels("endpoints", instance) }, { data = {
    HARVESTER_API_URL = "http://kubeoptix-harvester:8000", ANALYZER_API_URL = "http://kubeoptix-analyzer:8000",
    CORE_AI_API_URL = "http://kubeoptix-core-ai:8000", CONFIGURATIONS_API_URL = "http://kubeoptix-configurations:8000",
    REPORTER_API_URL = "http://kubeoptix-reporter:8000",
  }}))
  if not storage.existingClaim then
    local pvc_spec = { accessModes = { "ReadWriteMany" }, resources = { requests = { storage = storage.size or "10Gi" } } }
    if storage.storageClassName then pvc_spec.storageClassName = storage.storageClassName end
    table.insert(resources, object("v1", "PersistentVolumeClaim", { name = "kubeoptix-data", namespace = namespace, labels = labels("data", instance) }, pvc_spec))
  end

  table.insert(resources, object("v1", "ServiceAccount", { name = "kubeoptix-harvester", namespace = namespace, labels = labels("harvester", instance) }))
  table.insert(resources, object("rbac.authorization.k8s.io/v1", "ClusterRole", { name = "kubeoptix-harvester-" .. namespace }, { rules = {
    { apiGroups = { "" }, resources = { "nodes", "pods", "services", "configmaps", "events", "persistentvolumeclaims", "serviceaccounts" }, verbs = { "get", "list" } },
    { apiGroups = { "apps", "autoscaling", "batch" }, resources = { "deployments", "statefulsets", "daemonsets", "replicasets", "horizontalpodautoscalers", "jobs", "cronjobs" }, verbs = { "get", "list" } },
    { apiGroups = { "route.openshift.io", "monitoring.coreos.com" }, resources = { "routes", "servicemonitors", "podmonitors", "prometheusrules" }, verbs = { "get", "list" } },
  }}))
  table.insert(resources, object("rbac.authorization.k8s.io/v1", "ClusterRoleBinding", { name = "kubeoptix-harvester-" .. namespace }, { roleRef = { apiGroup = "rbac.authorization.k8s.io", kind = "ClusterRole", name = "kubeoptix-harvester-" .. namespace }, subjects = {{ kind = "ServiceAccount", name = "kubeoptix-harvester", namespace = namespace }} }))

  if enabled(spec, "configurations") then
    table.insert(resources, object("apps/v1", "StatefulSet", { name = "kubeoptix-postgresql", namespace = namespace, labels = labels("postgresql", instance) }, { serviceName = "kubeoptix-postgresql", replicas = 1, selector = { matchLabels = labels("postgresql", instance) }, template = { metadata = { labels = labels("postgresql", instance) }, spec = { containers = {{ name = "postgresql", image = images.postgres, env = { env_secret("POSTGRESQL_USER", secret_name, "POSTGRESQL_USER"), env_secret("POSTGRESQL_PASSWORD", secret_name, "POSTGRESQL_PASSWORD"), env_secret("POSTGRESQL_DATABASE", secret_name, "POSTGRESQL_DATABASE") }, ports = {{ name = "postgresql", containerPort = 5432 }}, volumeMounts = {{ name = "postgres-data", mountPath = "/var/lib/pgsql/data" }} }}, volumes = {{ name = "postgres-data", persistentVolumeClaim = { claimName = "kubeoptix-data" }}} }}}))
    table.insert(resources, object("v1", "Service", { name = "kubeoptix-postgresql", namespace = namespace, labels = labels("postgresql", instance) }, { selector = labels("postgresql", instance), ports = {{ name = "postgresql", port = 5432, targetPort = "postgresql" }} }))
    table.insert(resources, deployment("kubeoptix-configurations", namespace, instance, images.configurations, { env_secret("POSTGRESQL_USER", secret_name, "POSTGRESQL_USER"), env_secret("POSTGRESQL_PASSWORD", secret_name, "POSTGRESQL_PASSWORD"), env_secret("POSTGRESQL_DATABASE", secret_name, "POSTGRESQL_DATABASE"), { name = "POSTGRESQL_HOST", value = "kubeoptix-postgresql" }, { name = "QUARKUS_HIBERNATE_ORM_DATABASE_GENERATION", value = "update" } }))
    table.insert(resources, service("kubeoptix-configurations", namespace, instance))
  end
  local shared_mount = {{ name = "kubeoptix-data", mountPath = "/app/data" }}
  if enabled(spec, "harvester") then table.insert(resources, deployment("kubeoptix-harvester", namespace, instance, images.harvester, nil, shared_mount, data_claim)); table.insert(resources, service("kubeoptix-harvester", namespace, instance)) end
  if enabled(spec, "analyzer") then table.insert(resources, deployment("kubeoptix-analyzer", namespace, instance, images.analyzer, nil, shared_mount, data_claim)); table.insert(resources, service("kubeoptix-analyzer", namespace, instance)) end
  if enabled(spec, "coreAi") then table.insert(resources, deployment("kubeoptix-core-ai", namespace, instance, images.coreAi, nil, shared_mount, data_claim)); table.insert(resources, service("kubeoptix-core-ai", namespace, instance)) end
  if enabled(spec, "reporter") then table.insert(resources, deployment("kubeoptix-reporter", namespace, instance, images.reporter, {{ name = "CONFIGURATIONS_API_URL", value = "http://kubeoptix-configurations:8000" }}, shared_mount, data_claim)); table.insert(resources, service("kubeoptix-reporter", namespace, instance)) end
  if enabled(spec, "dashboard") then table.insert(resources, deployment("kubeoptix-dashboard", namespace, instance, images.dashboard, {{ name = "HARVESTER_API_URL", value = "http://kubeoptix-harvester:8000" }, { name = "ANALYZER_API_URL", value = "http://kubeoptix-analyzer:8000" }, { name = "CORE_AI_API_URL", value = "http://kubeoptix-core-ai:8000" }, { name = "SETTINGS_API_URL", value = "http://kubeoptix-configurations:8000" }, { name = "REPORTER_API_URL", value = "http://kubeoptix-reporter:8000" }}, nil)); table.insert(resources, service("kubeoptix-dashboard", namespace, instance)) end
  return resources, namespace
end

local function update_status(resource, namespace)
  local spec, metadata = resource.spec or {}, resource.metadata or {}
  local ready, desired = 0, 0
  for _, name in ipairs({ "harvester", "analyzer", "coreAi", "configurations", "reporter", "dashboard" }) do
    if enabled(spec, name) then
      desired = desired + 1
      local deployment_name = "kubeoptix-" .. (name == "coreAi" and "core-ai" or name)
      local deployment = get_json("-n " .. shell_quote(namespace) .. " get deployment " .. shell_quote(deployment_name))
      if deployment and deployment.status and deployment.status.readyReplicas and deployment.status.readyReplicas >= 1 then ready = ready + 1 end
    end
  end
  local phase = ready == desired and "Ready" or "Progressing"
  local status = cjson.encode({ status = { phase = phase, readyComponents = string.format("%d/%d", ready, desired), observedGeneration = metadata.generation or 0 } })
  local command = kubectl .. " -n " .. shell_quote(metadata.namespace) .. " patch shiftwise " .. shell_quote(metadata.name) .. " --subresource=status --type=merge -p " .. shell_quote(status)
  local ok, output = run(command)
  if not ok then log("error", "failed to update status", { name = metadata.name, output = output }) end
  return phase, ready, desired
end

local function reconcile(resource)
  local resources, namespace = build_resources(resource)
  local manifest = cjson.encode({ apiVersion = "v1", kind = "List", items = resources })
  local ok, output = run(kubectl .. " apply -f -", manifest)
  if not ok then error(output) end
  local phase, ready, desired = update_status(resource, namespace)
  log("info", "reconciled ShiftWise", { name = resource.metadata.name, namespace = namespace, phase = phase, ready = ready, desired = desired })
end

local function main()
  log("info", "starting ShiftWise Lua operator", { interval_seconds = poll_seconds })
  while true do
    local list, err = get_json("get shiftwises.shiftwise.ai --all-namespaces")
    if not list then
      log("error", "failed to list ShiftWise resources", { error = err })
    else
      for _, resource in ipairs(list.items or {}) do
        if not (resource.metadata and resource.metadata.deletionTimestamp) then
          local ok, reconcile_err = pcall(reconcile, resource)
          if not ok then log("error", "reconciliation failed", { name = resource.metadata and resource.metadata.name, error = reconcile_err }) end
        end
      end
    end
    os.execute("sleep " .. tonumber(poll_seconds))
  end
end

main()