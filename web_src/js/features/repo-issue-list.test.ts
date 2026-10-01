import {initIssueListCloseReason} from './repo-issue-list.ts';
import {createElementFromHTML} from '../utils/dom.ts';

// the markup the Issues tab's toolbar gets while open issues are listed
function createCloseButtons() {
  const elButtons = createElementFromHTML<HTMLElement>(`
    <div class="ui red basic buttons" data-global-init="initIssueListCloseReason">
      <button class="ui button issue-action" data-action="close" data-url="/user2/repo1/issues/status" data-close-reason="completed">Close as completed</button>
      <div class="ui dropdown icon button">
        <div class="menu">
          <div class="item js-aria-clickable selected" data-value="completed" data-status="Close as completed">Completed</div>
          <div class="item js-aria-clickable" data-value="not_planned" data-status="Close as not planned">Not planned</div>
        </div>
      </div>
    </div>`);
  initIssueListCloseReason(elButtons);
  const closeButton = elButtons.querySelector('.issue-action')!;
  const pick = (reason: string) => elButtons.querySelector<HTMLElement>(`.item[data-value="${CSS.escape(reason)}"]`)!.click();
  return {elButtons, closeButton, pick};
}

test('picking a reason gives the Close button that reason and its text', () => {
  const {closeButton, pick} = createCloseButtons();
  pick('not_planned');
  expect(closeButton.getAttribute('data-close-reason')).toEqual('not_planned');
  expect(closeButton.textContent).toEqual('Close as not planned');
  pick('completed'); // the reason can still be changed before closing
  expect(closeButton.getAttribute('data-close-reason')).toEqual('completed');
  expect(closeButton.textContent).toEqual('Close as completed');
});

test('a click in the menu that misses every item changes nothing', () => {
  const {elButtons, closeButton} = createCloseButtons();
  elButtons.querySelector<HTMLElement>('.menu')!.click();
  expect(closeButton.getAttribute('data-close-reason')).toEqual('completed');
  expect(closeButton.textContent).toEqual('Close as completed');
});
