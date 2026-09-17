<script lang="ts" setup>
import {SvgIcon} from '../svg.ts';
import {
  Chart,
  Legend,
  Tooltip,
  BarElement,
  LinearScale,
  TimeScale,
  PointElement,
  LineElement,
  type ChartOptions,
  type ChartData,
  type ChartDataset,
} from 'chart.js';
import {GET} from '../modules/fetch.ts';
import {Bar} from 'vue-chartjs';
import {chartJsColors} from '../utils/color.ts';
import {errorMessage} from '../modules/errors.ts';
import 'chartjs-adapter-dayjs-4/dist/chartjs-adapter-dayjs-4.esm';
import {computed, onMounted, shallowRef} from 'vue';

const {pageData} = window.config;

Chart.defaults.color = chartJsColors.text;
Chart.defaults.borderColor = chartJsColors.border;

Chart.register(
  TimeScale,
  LinearScale,
  Legend,
  Tooltip,
  BarElement,
  PointElement,
  LineElement,
);

const props = defineProps<{
  locale: {
    loadingTitle: string;
    loadingTitleFailed: string;
    loadingInfo: string;
    title: string;
    open: string;
    opened: string;
    closed: string;
    empty: string;
    rangeMonth: string;
    rangeQuarter: string;
    rangeYear: string;
    rangeAll: string;
  };
}>();

type IssueWeek = {
  week: number, // start of the week as a Unix timestamp in milliseconds
  open: number,
  opened: number,
  closed: number,
};

const weekMillis = 7 * 24 * 60 * 60 * 1000;

// number of weeks each range keeps, counting back from the latest one; 0 keeps every week
const ranges = [
  {key: 'month', weeks: 5},
  {key: 'quarter', weeks: 13},
  {key: 'year', weeks: 52},
  {key: 'all', weeks: 0},
] as const;

const rangeLabels: Record<string, keyof typeof props.locale> = {
  month: 'rangeMonth',
  quarter: 'rangeQuarter',
  year: 'rangeYear',
  all: 'rangeAll',
};

const isLoading = shallowRef(false);
const errorText = shallowRef('');
const repoLink = pageData.repoLink!;
const weeks = shallowRef<IssueWeek[]>([]);
const selectedRange = shallowRef<string>('quarter');

const visibleWeeks = computed<IssueWeek[]>(() => {
  const range = ranges.find((r) => r.key === selectedRange.value)!;
  if (!range.weeks || weeks.value.length === 0) return weeks.value;

  // a repository younger than the range has no weeks to show at the start of it, but the
  // axis should still span the range the reader asked for rather than collapsing onto the
  // few weeks that exist, which would also stretch the bars across the whole chart
  const shown = weeks.value.slice(-range.weeks);
  const padding: IssueWeek[] = [];
  for (let missing = range.weeks - shown.length; missing > 0; missing--) {
    padding.push({week: shown[0].week - missing * weekMillis, open: 0, opened: 0, closed: 0});
  }
  return [...padding, ...shown];
});

onMounted(() => {
  fetchGraphData();
});

async function fetchGraphData() {
  isLoading.value = true;
  try {
    const response = await GET(`${repoLink}/activity/issues/data`);
    if (response.ok) {
      // the endpoint answers null for a repository that has never had an issue
      weeks.value = (await response.json()) ?? [];
      errorText.value = '';
    } else {
      errorText.value = response.statusText;
    }
  } catch (err) {
    errorText.value = errorMessage(err);
  } finally {
    isLoading.value = false;
  }
}

function toGraphData(data: IssueWeek[]): ChartData<'bar'> {
  return {
    datasets: [
      {
        type: 'line',
        data: data.map((i) => ({x: i.week, y: i.open})),
        label: props.locale.open,
        borderColor: chartJsColors['commits'],
        backgroundColor: chartJsColors['commits'],
        pointRadius: 2,
        pointHitRadius: 6,
        borderWidth: 2,
        tension: 0.3,
        order: 0,
      } as unknown as ChartDataset<'bar'>,
      {
        data: data.map((i) => ({x: i.week, y: i.opened})),
        label: props.locale.opened,
        backgroundColor: chartJsColors['additions'],
        borderWidth: 0,
        order: 1,
      } as unknown as ChartDataset<'bar'>,
      {
        data: data.map((i) => ({x: i.week, y: i.closed})),
        label: props.locale.closed,
        backgroundColor: chartJsColors['deletions'],
        borderWidth: 0,
        order: 1,
      } as unknown as ChartDataset<'bar'>,
    ],
  };
}

const options: ChartOptions<'bar'> = {
  responsive: true,
  maintainAspectRatio: false,
  interaction: {
    mode: 'index',
    intersect: false,
  },
  plugins: {
    legend: {
      display: true,
    },
    tooltip: {
      callbacks: {
        // buckets are Sundays in UTC, so read them back in UTC or a viewer west of it sees
        // the Saturday before the week the numbers actually belong to
        title: (items) => new Date(items[0].parsed.x!).toLocaleDateString(undefined, {timeZone: 'UTC'}),
      },
    },
  },
  scales: {
    x: {
      type: 'time',
      // pad half a week at each end so the newest bar is not clipped by the axis
      offset: true,
      grid: {
        display: false,
      },
      time: {
        minUnit: 'week',
      },
      ticks: {
        maxRotation: 0,
        maxTicksLimit: 12,
      },
    },
    y: {
      beginAtZero: true,
      ticks: {
        maxTicksLimit: 6,
        precision: 0,
      },
    },
  },
} satisfies ChartOptions;
</script>

<template>
  <div>
    <div class="tw-flex tw-items-center tw-justify-between tw-flex-wrap tw-gap-2">
      <div class="ui header tw-mb-0">
        {{ isLoading ? locale.loadingTitle : errorText ? locale.loadingTitleFailed : locale.title }}
      </div>
      <div v-if="weeks.length !== 0" class="ui compact small menu">
        <a
          v-for="range in ranges" :key="range.key" class="item"
          :class="{active: selectedRange === range.key}"
          href="#" @click.prevent="selectedRange = range.key"
        >{{ locale[rangeLabels[range.key]] }}</a>
      </div>
    </div>
    <div class="tw-flex ui segment main-graph">
      <div v-if="isLoading || errorText !== '' || weeks.length === 0" class="tw-m-auto">
        <div v-if="isLoading">
          <SvgIcon name="gitea-running" class="tw-mr-2 rotate-clockwise"/>
          {{ locale.loadingInfo }}
        </div>
        <div v-else-if="errorText !== ''" class="tw-text-red">
          <SvgIcon name="octicon-x-circle-fill"/>
          {{ errorText }}
        </div>
        <div v-else class="tw-text-text-light">
          {{ locale.empty }}
        </div>
      </div>
      <Bar
        v-memo="[visibleWeeks]" v-if="visibleWeeks.length !== 0"
        :data="toGraphData(visibleWeeks)" :options="options"
      />
    </div>
  </div>
</template>

<style scoped>
.main-graph {
  height: 440px;
}
</style>
