// Draws the daisyUI ApexCharts charts on a page: every element with
// data-chart="<kind>" and its data as JSON in data-chart-data, with one
// <template id="chart-tooltip"> for the tooltips. Pages include
// it deferred after their charts; ChartsLayout loads ApexCharts deferred in
// <head>, so by document order ApexCharts runs first and the page is parsed.
// Colors are theme variables, so charts follow theme changes.
(() => {
  const baseContent = "var(--color-base-content)";
  const shade = (transparent) => `color-mix(in srgb, var(--color-base-content), transparent ${transparent}%)`;
  // tooltip fills the page's #chart-tooltip template; ApexCharts takes HTML, and
  // serializing the clone escapes the text.
  const tooltip = (dot, label, value) => {
    const tip = document.getElementById("chart-tooltip").content.firstElementChild.cloneNode(true);
    tip.querySelector("[data-tooltip-dot]").classList.add(dot);
    tip.querySelector("[data-tooltip-label]").textContent = label;
    tip.querySelector("[data-tooltip-value]").textContent = value;
    return tip.outerHTML;
  };

  // Column Chart One: monthly revenue columns.
  const columns = (data) => ({
    chart: { height: "300px", type: "bar", fontFamily: "inherit", toolbar: { show: false }, injectStyleSheet: false },
    series: [{ name: data.name, data: data.values }],
    colors: [baseContent],
    plotOptions: { bar: { horizontal: false, columnWidth: "55%", borderRadius: 5, borderRadiusApplication: "end" } },
    dataLabels: { enabled: false },
    stroke: { show: true, width: 2, colors: ["transparent"] },
    xaxis: {
      categories: data.labels,
      labels: { style: { colors: baseContent, fontSize: "12px" } },
      axisBorder: { show: true, color: shade(90) },
      axisTicks: { show: true, color: shade(90) },
    },
    yaxis: { show: false },
    fill: { opacity: 1 },
    grid: { borderColor: shade(95) },
    legend: { show: false },
    tooltip: {
      enabled: true,
      custom: ({ dataPointIndex }) => tooltip(data.dots[dataPointIndex], data.titles[dataPointIndex], data.formatted[dataPointIndex]),
    },
  });

  // Pie Chart Nine: share donut with the total in the middle.
  const donut = (data) => ({
    chart: { height: "220px", type: "donut", fontFamily: "inherit", toolbar: { show: false }, injectStyleSheet: false },
    series: data.values,
    labels: data.labels,
    colors: data.values.map(() => baseContent),
    fill: { opacity: data.opacities },
    stroke: { show: true, width: 1, colors: ["var(--color-base-200)"] },
    plotOptions: {
      pie: {
        expandOnClick: false,
        donut: {
          size: "62%",
          labels: {
            show: true,
            name: { show: true, fontSize: "11px", color: baseContent, offsetY: -2 },
            value: {
              show: true,
              fontSize: "18px",
              fontWeight: 700,
              color: baseContent,
              offsetY: 4,
              formatter: (_, opts) => data.formatted[opts.seriesIndex] ?? "",
            },
            total: { show: true, showAlways: true, label: data.totalLabel, fontSize: "11px", color: baseContent, formatter: () => data.total },
          },
        },
      },
    },
    dataLabels: { enabled: false },
    legend: { show: false },
    tooltip: {
      enabled: true,
      custom: ({ seriesIndex }) => tooltip(data.dots[seriesIndex], data.labels[seriesIndex], data.formatted[seriesIndex]),
    },
  });

  // Area Chart Thirteen: compact sparkline of a time series. data.times are
  // unix seconds, rendered as UTC clock times; data.suffix follows each value.
  const sparkline = (data) => {
    const clock = (seconds) => new Date(seconds * 1000).toISOString().slice(11, 19);
    const format = (value) => `${Number(value.toFixed(2)).toLocaleString()}${data.suffix}`;

    return {
      chart: {
        height: "160px",
        type: "area",
        fontFamily: "inherit",
        toolbar: { show: false },
        zoom: { enabled: false },
        dropShadow: { enabled: false },
        animations: { enabled: false },
        injectStyleSheet: false,
      },
      series: [{ name: data.name, data: data.values }],
      colors: [baseContent],
      stroke: { curve: "smooth", width: 2 },
      fill: {
        type: "gradient",
        gradient: { opacityFrom: 0.45, opacityTo: 0, shade: baseContent, gradientToColors: [baseContent] },
      },
      dataLabels: { enabled: false },
      grid: { show: false },
      legend: { show: false },
      xaxis: {
        categories: data.times.map(clock),
        labels: { show: false },
        axisBorder: { show: false },
        axisTicks: { show: false },
        tooltip: { enabled: false },
      },
      yaxis: { show: false },
      tooltip: {
        enabled: true,
        custom: ({ dataPointIndex }) => tooltip("bg-base-content", `${clock(data.times[dataPointIndex])} UTC`, format(data.values[dataPointIndex])),
      },
    };
  };

  const kinds = { columns, donut, sparkline };

  // Every chart element has one ApexCharts instance while it is in the page.
  // Pages that swap chart elements in and out (a polled fragment, a boosted
  // visit) are handled by watching the document: an element that is added gets
  // its chart, an element that is removed has its chart destroyed.
  const instances = new WeakMap();

  const chartsIn = (node) => {
    if (node.nodeType !== Node.ELEMENT_NODE) return [];
    const found = [...node.querySelectorAll("[data-chart]")];
    return node.matches("[data-chart]") ? [node, ...found] : found;
  };

  const mount = (el) => {
    const build = kinds[el.dataset.chart];
    if (!build || instances.has(el)) return;
    const chart = new window.ApexCharts(el, build(JSON.parse(el.dataset.chartData)));
    instances.set(el, chart);
    chart.render();
  };

  const unmount = (el) => {
    instances.get(el)?.destroy();
    instances.delete(el);
  };

  new MutationObserver((mutations) => {
    for (const { addedNodes, removedNodes } of mutations) {
      for (const node of removedNodes) chartsIn(node).forEach(unmount);
      for (const node of addedNodes) chartsIn(node).forEach(mount);
    }
  }).observe(document.documentElement, { childList: true, subtree: true });

  chartsIn(document.documentElement).forEach(mount);
})();
