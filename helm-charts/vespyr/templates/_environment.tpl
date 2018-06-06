{{- define "environment" -}}
- name: POSTGRES
  valueFrom:
    secretKeyRef:
      name: {{ .Values.secretKey }}
      key: postgres
- name: VESPYR_LOG_JSON
  value: "true"
- name: VESPYR_LOG
  value: "DEBUG"
- name: SLACK_TRADES_CHANNEL
  value: "{{ .Values.app.slackTradesChannel }}"
- name: SLACK_DATA_CHANNEL
  value: "{{ .Values.app.slackDataChannel }}"
- name: USE_FAKE_EXCHANGE
  value: "{{ .Values.app.useFakeExchange }}"
- name: SLACK_TOKEN
  valueFrom:
    secretKeyRef:
      name: {{ .Values.secretKey }}
      key: slackToken
- name: ENV
  value: {{ .Values.app.env }}
- name: ROLLBAR_TOKEN
  valueFrom:
    secretKeyRef:
      name: {{ .Values.secretKey }}
      key: rollbarToken
- name: GDAX_API_SECRET
  valueFrom:
    secretKeyRef:
      name: {{ .Values.secretKey }}
      key: gdaxAPISecret
- name: GDAX_API_KEY
  valueFrom:
    secretKeyRef:
      name: {{ .Values.secretKey }}
      key: gdaxAPIKey
- name: GDAX_PASSPHRASE
  valueFrom:
    secretKeyRef:
      name: {{ .Values.secretKey }}
      key: gdaxPassphrase
{{- end -}}