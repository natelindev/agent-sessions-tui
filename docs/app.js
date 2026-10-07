/* Progressive enhancements: all documentation is readable without JavaScript. */
document.documentElement.classList.add("js");

const menu = document.querySelector(".menu-toggle");
const sidebar = document.querySelector("#sidebar");
menu.addEventListener("click", () => {
	const open = menu.getAttribute("aria-expanded") !== "true";
	menu.setAttribute("aria-expanded", String(open));
	sidebar.classList.toggle("is-open", open);
});

document.addEventListener("keydown", (event) => {
	if (event.key === "Escape" && menu.getAttribute("aria-expanded") === "true") {
		menu.setAttribute("aria-expanded", "false");
		sidebar.classList.remove("is-open");
		menu.focus();
	}
});

const links = [...sidebar.querySelectorAll('a[href^="#"]')];
links.forEach((link) =>
	link.addEventListener("click", () => {
		menu.setAttribute("aria-expanded", "false");
		sidebar.classList.remove("is-open");
		const section = document.querySelector(link.getAttribute("href"));
		section.setAttribute("tabindex", "-1");
		section.focus({ preventScroll: true });
	}),
);

// Use the section nearest the top of the viewport, including long reference sections.
let scrollPending = false;
function updateActiveLink() {
	const threshold = 150;
	let active = links[0];
	links.forEach((link) => {
		if (
			document.querySelector(link.getAttribute("href")).getBoundingClientRect()
				.top <= threshold
		) {
			active = link;
		}
	});
	links.forEach((link) => {
		if (link === active) link.setAttribute("aria-current", "location");
		else link.removeAttribute("aria-current");
	});
	scrollPending = false;
}
window.addEventListener(
	"scroll",
	() => {
		if (!scrollPending) {
			scrollPending = true;
			window.requestAnimationFrame(updateActiveLink);
		}
	},
	{ passive: true },
);
updateActiveLink();

const status = document.querySelector("#copy-status");
document.querySelectorAll(".code-block").forEach((block, index) => {
	const code = block.querySelector("code");
	const button = document.createElement("button");
	button.type = "button";
	button.className = "copy-button";
	button.textContent = "Copy";
	button.setAttribute("aria-label", `Copy code example ${index + 1}`);
	button.addEventListener("click", async () => {
		try {
			await navigator.clipboard.writeText(code.textContent);
			button.textContent = "Copied";
			status.textContent = `Code example ${index + 1} copied to clipboard.`;
		} catch {
			const range = document.createRange();
			range.selectNodeContents(code);
			const selection = window.getSelection();
			selection.removeAllRanges();
			selection.addRange(range);
			button.textContent = "Selected";
			status.textContent =
				"Clipboard unavailable. Code selected; use your copy shortcut.";
		}
		window.setTimeout(() => {
			button.textContent = "Copy";
		}, 2000);
	});
	block.append(button);
});

const themeToggle = document.querySelector(".theme-toggle");
function setTheme(theme) {
  document.documentElement.dataset.theme = theme;
  themeToggle.setAttribute("aria-label", `Switch to ${theme === "dark" ? "light" : "dark"} theme`);
  themeToggle.textContent = theme === "dark" ? "Light" : "Dark";
}
try { setTheme(localStorage.getItem("docs-theme") || (matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light")); } catch { setTheme("light"); }
themeToggle.addEventListener("click", () => {
  const theme = document.documentElement.dataset.theme === "dark" ? "light" : "dark";
  setTheme(theme);
  try { localStorage.setItem("docs-theme", theme); } catch {}
});
