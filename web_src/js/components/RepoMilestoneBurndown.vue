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
  burndownDataUrl,
  burndownSummary,
  changeLink,
  changeTooltipLines,
  changesOn,
  formatLocale,
  idealLine,
  includePullsFromSearch,
  latestChangeDate,
  scopeMarkers,
  searchWithIncludePulls,
  projectionLine,
  type BurndownChange,
  type BurndownSummaryLocale,
  type LinePoint,
  type MarkerPoint,
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
  canReadPulls: boolean; // the server enforces this too; here it only hides a toggle that could do nothing
  locale: BurndownSummaryLocale & {
    loadingTitle: string;
    loadingTitleFailed: string;
    loadingInfo: string;
    title: string;
    remaining: string;
    scope: string;
    ideal: string;
    projection: string;
    scopeAdded: string;
    scopeRemoved: string;
    empty: string;
    noDeadline: string;
    includePulls: string;
    changesOn: string;
    changeClosed: string;
    changeReopened: string;
    changesMore: string;
  };
}>();

const isLoading = shallowRef(false);
const errorText = shallowRef('');
const burndown = shallowRef<MilestoneBurndown | null>(null);
const includePulls = shallowRef(props.canReadPulls && includePullsFromSearch(window.location.search));
const selectedDate = shallowRef('');

const hasChart = computed(() => burndown.value !== null && burndown.value.status !== 'empty');
const summary = computed(() => (hasChart.value ? burndownSummary(burndown.value!, props.locale) : ''));
// joined here, not in the template, whose compiler drops the space between two interpolations
const summaryLine = computed(() => (burndown.value?.deadline === '' ? `${summary.value} · ${props.locale.noDeadline}` : summary.value));
// built once per fetch, not per render, so moving the pointer over the chart does not redraw it
const chartData = computed(() => (hasChart.value ? toChartData(burndown.value!) : null));
const chartOptions = computed(() => (hasChart.value ? toChartOptions(burndown.value!) : null));
const selectedChanges = computed(() => (hasChart.value ? changesOn(burndown.value!, selectedDate.value) : []));

const changeLabels = computed<Record<BurndownChange['kind'], string>>(() => ({
  closed: props.locale.changeClosed,
  reopened: props.locale.changeReopened,
  added: props.locale.scopeAdded,
  removed: props.locale.scopeRemoved,
}));

onMounted(() => {
  fetchBurndown();
});

async function fetchBurndown() {
  isLoading.value = true;
  try {
    const response = await GET(burndownDataUrl(pageData.milestoneBurndownLink!, includePulls.value));
    if (response.ok) {
      burndown.value = await response.json();
      selectedDate.value = latestChangeDate(burndown.value!);
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

// keep the choice in the URL so a shared link shows the same chart, without touching the issue list's filters
function toggleIncludePulls(event: Event) {
  includePulls.value = (event.target as HTMLInputElement).checked;
  const {pathname, search, hash} = window.location;
  window.history.replaceState(window.history.state, '', `${pathname}${searchWithIncludePulls(search, includePulls.value)}${hash}`);
  fetchBurndown();
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
      // each point is the end of its day, so hold the previous value and change only on the day it changed
      stepped: 'before',
      order: 0,
    },
    {
      label: props.locale.scope,
      data: data.points.map((p) => ({x: p.date, y: p.scope})),
      borderColor: chartJsColors.textLight, // strong enough to read the scope markers against
      backgroundColor: chartJsColors.textLight,
      borderWidth: 1,
      borderDash: [4, 4],
      pointRadius: 0,
      pointHitRadius: 6,
      stepped: 'before',
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
    // scope changes ride on the scope line; up and down triangles tell them apart without colour
    {
      label: props.locale.scopeAdded,
      data: scopeMarkers(data, 'added'),
      showLine: false,
      pointStyle: 'triangle',
      pointRotation: 0,
      pointRadius: 6,
      pointHoverRadius: 7,
      borderColor: chartJsColors.text,
      backgroundColor: chartJsColors.text,
      order: -1, // drawn over the lines
    },
    {
      label: props.locale.scopeRemoved,
      data: scopeMarkers(data, 'removed'),
      showLine: false,
      pointStyle: 'triangle',
      pointRotation: 180,
      pointRadius: 6,
      pointHoverRadius: 7,
      borderColor: chartJsColors.text,
      backgroundColor: chartJsColors.text,
      order: -1,
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
      // snap to the nearest day and show every series and any scope change on that date
      mode: 'nearest',
      axis: 'x',
      intersect: false,
    },
    // the list under the chart follows the pointer; it stays on the last day hovered, so its links can be reached
    onHover: (_event, elements, chart) => {
      if (!elements.length) return;
      const {datasetIndex, index} = elements[0];
      selectedDate.value = (chart.data.datasets[datasetIndex].data[index] as unknown as LinePoint).x; // x is our date string
    },
    plugins: {
      legend: {
        display: true,
        labels: {
          usePointStyle: true,
          // each line as its dashed line rather than a filled box, each scope marker as its triangle
          generateLabels: (chart) => Chart.defaults.plugins.legend.labels.generateLabels(chart).map((item) => {
            const dataset = chart.data.datasets[item.datasetIndex!] as {showLine?: boolean, pointRotation?: number};
            const isMarker = dataset.showLine === false;
            return {...item, pointStyle: isMarker ? 'triangle' : 'line', rotation: isMarker ? dataset.pointRotation! : 0};
          }),
        },
      },
      tooltip: {
        callbacks: {
          // show the server's own date string, not one the browser re-derived in the viewer's zone
          title: (items) => (items[0].raw as {x: string}).x,
          // a marker's y is where it sits on the scope line; what it means is how many items changed
          label: (item) => {
            const count = (item.raw as Partial<MarkerPoint>).count;
            return `${item.dataset.label}: ${count === undefined ? item.formattedValue : count}`;
          },
          // hovering a day names what changed on it; the list under the chart repeats it with links
          footer: (items) => changeTooltipLines(changesOn(data, (items[0].raw as LinePoint).x), changeLabels.value, props.locale.changesMore),
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
  <details class="milestone-burndown" open>
    <summary class="tw-flex tw-items-center tw-justify-between tw-flex-wrap tw-gap-2 tw-cursor-pointer">
      <strong class="flex-text-inline">
        <!-- the caret replaces the browser's marker, which a flex summary loses -->
        <SvgIcon name="octicon-chevron-right" class="milestone-burndown-caret"/>
        {{ isLoading ? locale.loadingTitle : errorText ? locale.loadingTitleFailed : locale.title }}
      </strong>
      <span v-if="hasChart" class="milestone-burndown-summary">
        {{ summaryLine }}
      </span>
    </summary>
    <label v-if="canReadPulls" class="flex-text-inline tw-mt-2">
      <input class="milestone-burndown-include-pulls" type="checkbox" :checked="includePulls" @change="toggleIncludePulls">
      {{ locale.includePulls }}
    </label>
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
      <Line v-else :data="chartData!" :options="chartOptions!" :aria-label="summary"/>
    </div>
    <div v-if="selectedChanges.length" class="milestone-burndown-changes">
      <div class="tw-font-semibold">{{ formatLocale(locale.changesOn, selectedDate) }}</div>
      <ul class="tw-my-1 tw-pl-5">
        <li v-for="(change, i) in selectedChanges" :key="i">
          {{ changeLabels[change.kind] }}:
          <a :href="changeLink(pageData.repoLink!, change)">#{{ change.index }} {{ change.title }}</a>
        </li>
      </ul>
    </div>
  </details>
</template>

<style scoped>
.milestone-burndown > summary {
  list-style: none;
}

.milestone-burndown > summary::-webkit-details-marker {
  display: none;
}

.milestone-burndown-caret {
  transition: transform 0.15s ease;
}

.milestone-burndown[open] .milestone-burndown-caret {
  transform: rotate(90deg);
}

.burndown-graph {
  height: 260px;
}
</style>
