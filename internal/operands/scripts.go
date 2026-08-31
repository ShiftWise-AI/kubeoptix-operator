package operands

const harvesterStartupScript = `echo "[INFO] Authenticating with in-cluster service account"
oc login https://kubernetes.default.svc:443 \
  --token="$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)" \
  --certificate-authority=/var/run/secrets/kubernetes.io/serviceaccount/ca.crt \
  && \
echo "[INFO] Starting API process" && \
exec /app/.venv/bin/python /app/src/api.py
`
