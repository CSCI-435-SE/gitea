<script lang="ts" setup>
import {SvgIcon} from '../svg.ts';
import {
  Chart,
  Legend,
  Tooltip,
  LinearScale,
  TimeScale,
  PointElement,
  LineElement,
  type ChartOptions,
  type ChartData,
  type ChartDataset,
} from 'chart.js';
import {GET} from '../modules/fetch.ts';
import {Line} from 'vue-chartjs';
import {chartJsColors} from '../utils/color.ts';
import {errorMessage} from '../modules/errors.ts';
import 'chartjs-adapter-dayjs-4/dist/chartjs-adapter-dayjs-4.esm';
import {computed, onMounted, shallowRef} from 'vue';
import {
  burndownAxisMax,
  burndownSummary,
  idealLine,
  projectionLine,
  type BurndownSummaryLocale,
  type LinePoint,
  type MilestoneBurndown,
} from './RepoMilestoneBurndown.utils.ts';

const {pageData} = window.config;

Chart.defaults.color = chartJsColors.text;
Chart.defaults.borderColor = chartJsColors.border;

Chart.register(
  TimeScale,
  LinearScale,
  Legend,
  Tooltip,
  PointElement,
  LineElement,
);

const props = defineProps<{
  locale: BurndownSummaryLocale & {
    loadingTitle: string;
    loadingTitleFailed: string;
    loadingInfo: string;
    title: string;
    remaining: string;
    scope: string;
    ideal: string;
    projection: string;
    empty: string;
    noDeadline: string;
  };
}>();

const isLoading = shallowRef(false);
const errorText = shallowRef('');
const burndown = shallowRef<MilestoneBurndown | null>(null);

const hasChart = computed(() => burndown.value !== null && burndown.value.status !== 'empty');
const summary = computed(() => (hasChart.value ? burndownSummary(burndown.value!, props.locale) : ''));

onMounted(() => {
  fetchBurndown();
});

async function fetchBurndown() {
  isLoading.value = true;
  try {
    const response = await GET(pageData.milestoneBurndownLink!);
    if (response.ok) {
      burndown.value = await response.json();
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

function toChartData(data: MilestoneBurndown): ChartData<'line'> {
  // dash patterns tell the lines apart without relying on their colours
  const late = data.daysLate !== null && data.daysLate > 0;
  const datasets: ChartDataset<'line', LinePoint[]>[] = [
    {
      label: props.locale.remaining,
      data: data.points.map((p) => ({x: p.date, y: p.remaining})),
      borderColor: chartJsColors.commits,
      backgroundColor: chartJsColors.commits,
      borderWidth: 2,
      pointRadius: data.points.length > 60 ? 0 : 2,
      pointHitRadius: 6,
      stepped: 'after', // work drops when an item closes, not gradually between closes
      order: 0,
    },
    {
      label: props.locale.scope,
      data: data.points.map((p) => ({x: p.date, y: p.scope})),
      borderColor: chartJsColors.border,
      backgroundColor: chartJsColors.border,
      borderWidth: 1,
      borderDash: [4, 4],
      pointRadius: 0,
      pointHitRadius: 6,
      stepped: 'after',
      order: 1,
    },
    {
      label: props.locale.ideal,
      data: idealLine(data),
      borderColor: chartJsColors.text,
      backgroundColor: chartJsColors.text,
      borderWidth: 1,
      borderDash: [6, 4],
      pointRadius: 0,
      pointHitRadius: 0,
      order: 2,
    },
    {
      label: props.locale.projection,
      data: projectionLine(data),
      borderColor: late ? chartJsColors.deletions : chartJsColors.additions,
      backgroundColor: late ? chartJsColors.deletions : chartJsColors.additions,
      borderWidth: 2,
      borderDash: [2, 3],
      pointRadius: 0,
      pointHitRadius: 6,
      order: 3,
    },
  ];
  // x is the server's date string, which the time scale parses; vue-chartjs types x as a number only
  return {datasets: datasets.filter((dataset) => dataset.data.length !== 0)} as unknown as ChartData<'line'>;
}

function toChartOptions(data: MilestoneBurndown): ChartOptions<'line'> {
  return {
    responsive: true,
    maintainAspectRatio: false,
    animation: false,
    interaction: {
      mode: 'nearest',
      intersect: false,
    },
    plugins: {
      legend: {
        display: true,
        // draw each entry as its line, dashes included, rather than as a filled box
        labels: {usePointStyle: true, pointStyle: 'line'},
      },
      tooltip: {
        callbacks: {
          // show the server's own date string, not one the browser re-derived in the viewer's zone
          title: (items) => (items[0].raw as {x: string}).x,
        },
      },
    },
    scales: {
      x: {
        type: 'time',
        min: data.points[0].date,
        max: burndownAxisMax(data),
        grid: {
          display: false,
        },
        time: {
          unit: 'day',
          parser: 'YYYY-MM-DD',
        },
        ticks: {
          maxRotation: 0,
          maxTicksLimit: 10,
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
  };
}
</script>

<template>
  <div class="milestone-burndown">
    <div class="tw-flex tw-items-baseline tw-justify-between tw-flex-wrap tw-gap-2">
      <strong>
        {{ isLoading ? locale.loadingTitle : errorText ? locale.loadingTitleFailed : locale.title }}
      </strong>
      <span v-if="hasChart" class="milestone-burndown-summary">
        {{ summary }}<template v-if="!burndown!.deadline"> {{ locale.noDeadline }}</template>
      </span>
    </div>
    <div class="tw-flex ui segment burndown-graph">
      <div v-if="isLoading || errorText !== '' || !hasChart" class="tw-m-auto">
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
      <!-- the canvas already has role="img"; label it with the summary so its verdict is not only visual -->
      <Line v-else :data="toChartData(burndown!)" :options="toChartOptions(burndown!)" :aria-label="summary"/>
    </div>
  </div>
</template>

<style scoped>
.burndown-graph {
  height: 260px;
}
</style>
