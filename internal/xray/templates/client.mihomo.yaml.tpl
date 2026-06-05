proxies:
  - name: "{{ .Name }}"
    type: vless
    server: {{ .ServerAddress }}
    port: {{ .ServerPort }}
    uuid: {{ .UUID }}
    network: tcp
    tls: true
    udp: true
    flow: xtls-rprx-vision
    servername: {{ .RealityServerName }}
    client-fingerprint: {{ .ClientFingerprint }}
    reality-opts:
      public-key: {{ .RealityPublicKey }}
      short-id: {{ .RealityShortID }}

proxy-groups:
  - name: PROXY
    type: select
    proxies:
      - "{{ .Name }}"

rules:
  - MATCH,PROXY
