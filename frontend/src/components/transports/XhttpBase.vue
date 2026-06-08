<template>
  <v-row>
    <v-col cols="12" sm="6" md="4">
      <v-combobox :label="$t('transport.host')" :items="sniFrontHosts" hide-details v-model="data.host" />
    </v-col>
    <v-col cols="12" sm="6" md="4">
      <v-combobox :label="$t('transport.path')" :items="pathPresets" hide-details v-model="data.path" />
    </v-col>
    <v-col cols="12" sm="6" md="4">
      <v-select
        :label="$t('transport.xhttpDomainStrategy')"
        :items="['','prefer_ipv4','prefer_ipv6','ipv4_only','ipv6_only']"
        clearable hide-details
        v-model="data.domain_strategy" />
    </v-col>
  </v-row>
  <v-row>
    <v-col cols="12" sm="6" md="4">
      <v-text-field :label="$t('transport.xhttpPaddingBytes')" placeholder="100-1000" hide-details
        :model-value="data.x_padding_bytes" @update:model-value="(v:string) => setRange('x_padding_bytes', v)" />
    </v-col>
    <v-col cols="12" sm="6" md="4">
      <v-text-field :label="$t('transport.xhttpScMaxEachPostBytes')" placeholder="1000000" hide-details
        :model-value="data.sc_max_each_post_bytes" @update:model-value="(v:string) => setRange('sc_max_each_post_bytes', v)" />
    </v-col>
    <v-col cols="12" sm="6" md="4">
      <v-text-field :label="$t('transport.xhttpScMinPostsIntervalMs')" placeholder="30" hide-details
        :model-value="data.sc_min_posts_interval_ms" @update:model-value="(v:string) => setRange('sc_min_posts_interval_ms', v)" />
    </v-col>
  </v-row>
  <v-row>
    <v-col cols="12" sm="6" md="4">
      <v-text-field :label="$t('transport.xhttpScStreamUpServerSecs')" placeholder="20-80" hide-details
        :model-value="data.sc_stream_up_server_secs" @update:model-value="(v:string) => setRange('sc_stream_up_server_secs', v)" />
    </v-col>
    <v-col cols="6" sm="4" md="3">
      <v-text-field :label="$t('transport.xhttpScMaxBufferedPosts')" type="number" min="0" hide-details
        :model-value="data.sc_max_buffered_posts" @update:model-value="(v:string) => setNum('sc_max_buffered_posts', v)" />
    </v-col>
    <v-col cols="6" sm="4" md="3">
      <v-text-field :label="$t('transport.xhttpServerMaxHeaderBytes')" type="number" min="0" hide-details
        :model-value="data.server_max_header_bytes" @update:model-value="(v:string) => setNum('server_max_header_bytes', v)" />
    </v-col>
  </v-row>
  <v-row>
    <v-col cols="6" sm="4" md="3" align-self="center">
      <v-switch color="primary" :label="$t('transport.xhttpNoGrpcHeader')" hide-details v-model="data.no_grpc_header" />
    </v-col>
    <v-col cols="6" sm="4" md="3" align-self="center">
      <v-switch color="primary" :label="$t('transport.xhttpNoSseHeader')" hide-details v-model="data.no_sse_header" />
    </v-col>
    <v-col cols="12" sm="6" md="6">
      <v-combobox chips multiple clearable hide-details
        :label="$t('transport.xhttpTrustedXff')"
        :model-value="data.trusted_x_forwarded_for"
        @update:model-value="setTrustedXff" />
    </v-col>
  </v-row>

  <v-expansion-panels variant="accordion" style="margin-top: 8px;">
    <v-expansion-panel :title="$t('transport.xhttpPaddingObfs')">
      <v-expansion-panel-text>
        <v-row>
          <v-col cols="12" sm="4" align-self="center">
            <v-switch color="primary" :label="$t('transport.xhttpPaddingObfsMode')" hide-details v-model="data.x_padding_obfs_mode" />
          </v-col>
          <v-col cols="12" sm="4">
            <v-text-field :label="$t('transport.xhttpPaddingKey')" placeholder="x_padding" hide-details v-model="data.x_padding_key" />
          </v-col>
          <v-col cols="12" sm="4">
            <v-text-field :label="$t('transport.xhttpPaddingHeader')" placeholder="X-Padding" hide-details v-model="data.x_padding_header" />
          </v-col>
          <v-col cols="12" sm="4">
            <v-select :label="$t('transport.xhttpPaddingPlacement')" :items="['','queryInHeader','cookie','header','query']" clearable hide-details v-model="data.x_padding_placement" />
          </v-col>
          <v-col cols="12" sm="4">
            <v-select :label="$t('transport.xhttpPaddingMethod')" :items="['','repeat-x','tokenish']" clearable hide-details v-model="data.x_padding_method" />
          </v-col>
        </v-row>
      </v-expansion-panel-text>
    </v-expansion-panel>
    <v-expansion-panel :title="$t('transport.xhttpPlacement')">
      <v-expansion-panel-text>
        <v-row>
          <v-col cols="12" sm="4">
            <v-select :label="$t('transport.xhttpUplinkHttpMethod')" :items="['','POST','GET']" clearable hide-details v-model="data.uplink_http_method" />
          </v-col>
          <v-col cols="12" sm="4">
            <v-select :label="$t('transport.xhttpSessionPlacement')" :items="['','path','cookie','header','query']" clearable hide-details v-model="data.session_placement" />
          </v-col>
          <v-col cols="12" sm="4">
            <v-text-field :label="$t('transport.xhttpSessionKey')" hide-details v-model="data.session_key" />
          </v-col>
          <v-col cols="12" sm="4">
            <v-select :label="$t('transport.xhttpSeqPlacement')" :items="['','path','cookie','header','query']" clearable hide-details v-model="data.seq_placement" />
          </v-col>
          <v-col cols="12" sm="4">
            <v-text-field :label="$t('transport.xhttpSeqKey')" hide-details v-model="data.seq_key" />
          </v-col>
          <v-col cols="12" sm="4">
            <v-select :label="$t('transport.xhttpUplinkDataPlacement')" :items="['','auto','body','cookie','header']" clearable hide-details v-model="data.uplink_data_placement" />
          </v-col>
          <v-col cols="12" sm="4">
            <v-text-field :label="$t('transport.xhttpUplinkDataKey')" hide-details v-model="data.uplink_data_key" />
          </v-col>
          <v-col cols="12" sm="4">
            <v-text-field :label="$t('transport.xhttpUplinkChunkSize')" hide-details
              :model-value="data.uplink_chunk_size" @update:model-value="(v:string) => setRange('uplink_chunk_size', v)" />
          </v-col>
        </v-row>
      </v-expansion-panel-text>
    </v-expansion-panel>
    <v-expansion-panel :title="$t('transport.xhttpXmux')">
      <v-expansion-panel-text>
        <v-row>
          <v-col cols="6" sm="4">
            <v-text-field :label="$t('transport.xmuxMaxConcurrency')" placeholder="16-32" hide-details
              :model-value="xmux.max_concurrency" @update:model-value="(v:string) => setXmux('max_concurrency', v)" />
          </v-col>
          <v-col cols="6" sm="4">
            <v-text-field :label="$t('transport.xmuxMaxConnections')" placeholder="0" hide-details
              :model-value="xmux.max_connections" @update:model-value="(v:string) => setXmux('max_connections', v)" />
          </v-col>
          <v-col cols="6" sm="4">
            <v-text-field :label="$t('transport.xmuxCMaxReuseTimes')" placeholder="0" hide-details
              :model-value="xmux.c_max_reuse_times" @update:model-value="(v:string) => setXmux('c_max_reuse_times', v)" />
          </v-col>
          <v-col cols="6" sm="4">
            <v-text-field :label="$t('transport.xmuxHMaxRequestTimes')" placeholder="600-900" hide-details
              :model-value="xmux.h_max_request_times" @update:model-value="(v:string) => setXmux('h_max_request_times', v)" />
          </v-col>
          <v-col cols="6" sm="4">
            <v-text-field :label="$t('transport.xmuxHMaxReusableSecs')" placeholder="1800-3000" hide-details
              :model-value="xmux.h_max_reusable_secs" @update:model-value="(v:string) => setXmux('h_max_reusable_secs', v)" />
          </v-col>
          <v-col cols="6" sm="4">
            <v-text-field :label="$t('transport.xmuxHKeepAlivePeriod')" type="number" min="0" hide-details
              :model-value="xmux.h_keep_alive_period" @update:model-value="(v:string) => setXmuxNum('h_keep_alive_period', v)" />
          </v-col>
        </v-row>
      </v-expansion-panel-text>
    </v-expansion-panel>
  </v-expansion-panels>

  <Headers :data="data" />
</template>

<script lang="ts">
import Headers from '../Headers.vue'
import { sniFrontHosts, RECOMMENDED } from '@/types/recommended'
export default {
  props: ['data'],
  data() {
    return {
      sniFrontHosts,
      pathPresets: [RECOMMENDED.wsPath],
    }
  },
  computed: {
    xmux(): any {
      return this.$props.data.xmux ?? {}
    },
  },
  methods: {
    setTrustedXff(v: string[]) {
      if (v && v.length > 0) {
        this.$props.data.trusted_x_forwarded_for = v
      } else {
        delete this.$props.data.trusted_x_forwarded_for
      }
    },
    // Range fields have no omitempty and the core rejects empty-string ranges,
    // so only persist the key while it holds a value.
    setRange(key: string, v: string) {
      if (v != null && v !== '') {
        this.$props.data[key] = v
      } else {
        delete this.$props.data[key]
      }
    },
    setNum(key: string, v: string) {
      if (v != null && v !== '') {
        this.$props.data[key] = Number(v)
      } else {
        delete this.$props.data[key]
      }
    },
    // Range fields have no omitempty and the core rejects empty-string ranges,
    // so only persist a key while it holds a value and drop the xmux object when empty.
    setXmux(key: string, v: string) {
      if (!this.$props.data.xmux) this.$props.data.xmux = {}
      if (v != null && v !== '') {
        this.$props.data.xmux[key] = v
      } else {
        delete this.$props.data.xmux[key]
      }
      if (Object.keys(this.$props.data.xmux).length === 0) delete this.$props.data.xmux
    },
    setXmuxNum(key: string, v: string) {
      if (!this.$props.data.xmux) this.$props.data.xmux = {}
      if (v != null && v !== '') {
        this.$props.data.xmux[key] = Number(v)
      } else {
        delete this.$props.data.xmux[key]
      }
      if (Object.keys(this.$props.data.xmux).length === 0) delete this.$props.data.xmux
    },
  },
  components: { Headers },
}
</script>
