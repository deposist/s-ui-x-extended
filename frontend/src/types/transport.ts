export const TrspTypes = {
  HTTP: 'http',
  WebSocket: 'ws',
  gRPC: 'grpc',
  HTTPUpgrade: "httpupgrade",
  XHTTP: 'xhttp',
  mKCP: 'mkcp',
  QUIC: 'quic',
}

export type TrspType = typeof TrspTypes[keyof typeof TrspTypes]

export type Transport = HTTP|WebSocket|gRPC|HTTPUpgrade|XHTTP|mKCP|QUIC

interface TransportBasics {
  type: TrspType
}

export interface HTTP extends TransportBasics {
  host?: string[]
  path?: string
  method?: string
  headers?: {}
  idle_timeout?: string
  ping_timeout?: string
}

export interface WebSocket extends TransportBasics {
  path: string
  headers?: {
    Host: string
  }
  max_early_data?: number
  early_data_header_name?: string
}

export interface gRPC extends TransportBasics {
  service_name?: string
  idle_timeout?: string
  ping_timeout?: string
  permit_without_stream?: boolean
}

export interface HTTPUpgrade extends TransportBasics {
  host?: string
  path?: string
  headers?: {}
}

export interface mKCP extends TransportBasics {
  mtu?: number
  tti?: number
  uplink_capacity?: number
  downlink_capacity?: number
  congestion?: boolean
  read_buffer_size?: number
  write_buffer_size?: number
  header_type?: string
  seed?: string
}

export interface QUIC extends TransportBasics {
  security?: string
  key?: string
  header?: {
    type?: string
  }
}

export interface XHTTPXmux {
  max_concurrency?: string
  max_connections?: string
  c_max_reuse_times?: string
  h_max_request_times?: string
  h_max_reusable_secs?: string
  h_keep_alive_period?: number
}

export interface XHTTPBase {
  host?: string
  path?: string
  headers?: {}
  domain_strategy?: string
  x_padding_bytes?: string
  no_grpc_header?: boolean
  no_sse_header?: boolean
  sc_max_each_post_bytes?: string
  sc_min_posts_interval_ms?: string
  sc_max_buffered_posts?: number
  sc_stream_up_server_secs?: string
  server_max_header_bytes?: number
  trusted_x_forwarded_for?: string[]
  xmux?: XHTTPXmux
  x_padding_obfs_mode?: boolean
  x_padding_key?: string
  x_padding_header?: string
  x_padding_placement?: string
  x_padding_method?: string
  uplink_http_method?: string
  session_placement?: string
  session_key?: string
  seq_placement?: string
  seq_key?: string
  uplink_data_placement?: string
  uplink_data_key?: string
  uplink_chunk_size?: string
}

export interface XHTTPDownload extends XHTTPBase {
  server?: string
  server_port?: number
  tls?: {}
  detour?: string
}

export interface XHTTP extends TransportBasics, XHTTPBase {
  mode?: string
  download?: XHTTPDownload
}
