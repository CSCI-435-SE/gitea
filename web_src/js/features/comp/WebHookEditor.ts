import {POST} from '../../modules/fetch.ts';
import {hideElem, showElem, toggleElem} from '../../utils/dom.ts';

export function initCompWebHookEditor() {
  if (!document.querySelectorAll('.new.webhook').length) {
    return;
  }

  for (const input of document.querySelectorAll<HTMLInputElement>('.events.checkbox input')) {
    input.addEventListener('change', function () {
      if (this.checked) {
        showElem('.events.fields');
      }
    });
  }

  for (const input of document.querySelectorAll<HTMLInputElement>('.non-events.checkbox input')) {
    input.addEventListener('change', function () {
      if (this.checked) {
        hideElem('.events.fields');
      }
    });
  }

  // some webhooks (like Gitea) allow to set the request method (GET/POST), and it would toggle the "Content Type" field
  const httpMethodInput = document.querySelector<HTMLInputElement>('#http_method');
  if (httpMethodInput) {
    const updateContentType = function () {
      const visible = httpMethodInput.value === 'POST';
      toggleElem(document.querySelector('#content_type')!.closest('.field')!, visible);
    };
    updateContentType();
    httpMethodInput.addEventListener('change', updateContentType);
  }

  // Test delivery, either a fake push or a ping
  for (const button of document.querySelectorAll<HTMLButtonElement>('.webhook-test-delivery')) {
    button.addEventListener('click', async () => {
      button.classList.add('is-loading', 'disabled');
      await POST(button.getAttribute('data-link')!);
      setTimeout(() => window.location.reload(), 5000);
    });
  }
}
