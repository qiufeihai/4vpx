{
  "log": {
    "loglevel": "{{ .XrayLogLevel }}"
  },
  "inbounds": [
    {
      "listen": "0.0.0.0",
      "port": {{ .ServerPort }},
      "protocol": "vless",
      "settings": {
        "clients": [
{{- range $index, $client := .Clients }}
          {{- if $index }},{{ end }}
          {
            "id": "{{ $client.UUID }}",
            "flow": "xtls-rprx-vision"
          }
{{- end }}
        ],
        "decryption": "none"
      },
      "streamSettings": {
        "network": "tcp",
        "security": "reality",
        "realitySettings": {
          "show": false,
          "dest": "{{ .RealityDest }}",
          "xver": 0,
          "serverNames": [
            "{{ .RealityServerName }}"
          ],
          "privateKey": "{{ .RealityPrivateKey }}",
          "shortIds": [
            "{{ .RealityShortID }}"
          ]
        }
      },
      "sniffing": {
        "enabled": true,
        "destOverride": [
          "http",
          "tls",
          "quic"
        ],
        "routeOnly": true
      }
    }
  ],
  "outbounds": [
    {
      "protocol": "freedom",
      "tag": "direct"
    }
  ]
}
