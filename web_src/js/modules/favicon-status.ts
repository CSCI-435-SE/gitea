import {getActionStatusIcon} from './action-status-icon.ts';
import type {ActionsStatus} from './gitea-actions.ts';
import {svgParseOuterInner} from '../svg.ts';
import {html, htmlRaw} from '../utils/html.ts';

const {svgOuter, svgInnerHtml: giteaFaviconInner} = svgParseOuterInner('gitea-favicon');
const faviconViewBox = svgOuter.getAttribute('viewBox')!;
const [, , faviconViewBoxWidth, faviconViewBoxHeight] = faviconViewBox.split(/\s+/).map(Number);

// the status badge is rendered in the bottom-right corner, following GitHub Actions favicon proportions
const badgeIconSize = 16;
const badgeSizeRatio = 340 / 640;
const badgeMargin = 6;
const badgeDrawSize = faviconViewBoxWidth * badgeSizeRatio;
const badgeX = faviconViewBoxWidth - badgeDrawSize - badgeMargin;
const badgeY = faviconViewBoxHeight - badgeDrawSize - badgeMargin;
const badgeScale = badgeDrawSize / badgeIconSize;
// white ring behind the badge so it stands out from the logo, like GitHub's favicon
const badgeCenter = badgeDrawSize / 2;
const badgeRingRadius = badgeCenter + badgeDrawSize * 0.08;

// two badges compete for the icon: an Actions run's status wins over the unread notification dot
let actionStatus: ActionsStatus | null = null;
let notificationDot = false;
let notificationDotDataUrl: string | null = null;
const defaultFaviconHrefs = new Map<HTMLLinkElement, string>();
const faviconDataUrlCache = new Map<ActionsStatus, string>();
let colorProbe: HTMLElement | null = null;

function rememberDefaultFaviconHrefs() {
  if (defaultFaviconHrefs.size > 0) return;
  for (const link of document.querySelectorAll<HTMLLinkElement>('link[rel~="icon"]')) {
    defaultFaviconHrefs.set(link, link.href);
  }
}

function resolveTailwindTextColor(colorClass: string): string {
  if (!colorProbe) {
    colorProbe = document.createElement('span');
    colorProbe.style.display = 'none';
    document.body.append(colorProbe);
  }
  colorProbe.className = colorClass;
  return getComputedStyle(colorProbe).color || '#000000';
}

function buildStatusIconMarkup(status: ActionsStatus): string {
  const {name, colorClass} = getActionStatusIcon(status, 'circle-fill');
  const color = resolveTailwindTextColor(colorClass);
  const {svgInnerHtml} = svgParseOuterInner(name);
  const coloredInner = svgInnerHtml.replaceAll('currentColor', color);
  const ring = html`<circle cx="${badgeX + badgeCenter}" cy="${badgeY + badgeCenter}" r="${badgeRingRadius}" fill="#ffffff"/>`;
  const badge = html`<g data-actions-status-name="${status}" transform="translate(${badgeX}, ${badgeY}) scale(${badgeScale})" fill="${color}" color="${color}">${htmlRaw(coloredInner)}</g>`;
  return html`${htmlRaw(ring)}${htmlRaw(badge)}`;
}

export function buildStatusFaviconSvg(status: ActionsStatus): string {
  return html`<svg xmlns="http://www.w3.org/2000/svg" viewBox="${faviconViewBox}">${htmlRaw(giteaFaviconInner)}${htmlRaw(buildStatusIconMarkup(status))}</svg>`;
}

function buildStatusFaviconDataUrl(status: ActionsStatus): string {
  const cached = faviconDataUrlCache.get(status);
  if (cached) return cached;
  const dataUrl = `data:image/svg+xml,${encodeURIComponent(buildStatusFaviconSvg(status))}`;
  faviconDataUrlCache.set(status, dataUrl);
  return dataUrl;
}

function setFaviconHref(href: string) {
  rememberDefaultFaviconHrefs();
  for (const link of defaultFaviconHrefs.keys()) {
    if (link.isConnected) link.href = href;
  }
}

export function buildNotificationFaviconSvg(): string {
  // same color as the navbar's unread count bubble
  const color = getComputedStyle(document.documentElement).getPropertyValue('--color-primary').trim() || '#4183c4';
  const cx = badgeX + badgeCenter;
  const cy = badgeY + badgeCenter;
  const ring = html`<circle cx="${cx}" cy="${cy}" r="${badgeRingRadius}" fill="#ffffff"/>`;
  const dot = html`<circle data-notification-dot="" cx="${cx}" cy="${cy}" r="${badgeCenter}" fill="${color}"/>`;
  return html`<svg xmlns="http://www.w3.org/2000/svg" viewBox="${faviconViewBox}">${htmlRaw(giteaFaviconInner)}${htmlRaw(ring)}${htmlRaw(dot)}</svg>`;
}

function buildNotificationFaviconDataUrl(): string {
  notificationDotDataUrl ??= `data:image/svg+xml,${encodeURIComponent(buildNotificationFaviconSvg())}`;
  return notificationDotDataUrl;
}

function restoreDefaultFavicon() {
  rememberDefaultFaviconHrefs();
  for (const [link, href] of defaultFaviconHrefs) {
    if (link.isConnected) link.href = href;
  }
}

function renderFavicon() {
  if (actionStatus) {
    setFaviconHref(buildStatusFaviconDataUrl(actionStatus));
  } else if (notificationDot) {
    setFaviconHref(buildNotificationFaviconDataUrl());
  } else {
    restoreDefaultFavicon();
  }
}

export function syncActionRunFavicon(status: ActionsStatus | ''): void {
  if (status === '') {
    resetActionFavicon();
    return;
  }
  if (status === actionStatus) return;
  actionStatus = status;
  renderFavicon();
}

export function resetActionFavicon(): void {
  if (actionStatus === null) return;
  actionStatus = null;
  renderFavicon();
}

export function syncNotificationFaviconDot(show: boolean): void {
  if (show === notificationDot) return;
  notificationDot = show;
  renderFavicon();
}
