// caetano / saldos-esperados — lista e importa ficheiros CSV da pasta Dropbox
// configurada, na página de importação Dropbox (data_dropbox.html).
(function () {
  "use strict";

  document.addEventListener("DOMContentLoaded", function () {
    var loading = document.getElementById("loading-state");
    var empty = document.getElementById("empty-state");
    var tableContainer = document.getElementById("table-container");
    var tbody = document.getElementById("file-list-body");
    var status = document.getElementById("status-text");
    var errorContainer = document.getElementById("error-container");
    var overlay = document.getElementById("progress-overlay");
    var processingName = document.getElementById("processing-filename");

    document.getElementById("btn-refresh").addEventListener("click", listFiles);
    listFiles();

    async function listFiles() {
      loading.hidden = false;
      empty.hidden = true;
      tableContainer.hidden = true;
      errorContainer.hidden = true;
      status.textContent = "A atualizar...";

      try {
        var resp = await fetch("/saldos-esperados/data/dropbox/list");
        var data = await resp.json();
        if (!resp.ok) throw new Error(data.error || "Erro ao listar ficheiros");

        loading.hidden = true;

        if (!data.files || data.files.length === 0) {
          empty.hidden = false;
          status.textContent = "0 ficheiros encontrados";
          return;
        }

        tbody.innerHTML = "";
        data.files.forEach(function (f) {
          var tr = document.createElement("tr");

          var tdName = document.createElement("td");
          tdName.textContent = f.name;
          tr.appendChild(tdName);

          var tdSize = document.createElement("td");
          tdSize.className = "cae-text-muted";
          tdSize.textContent = formatSize(f.size);
          tr.appendChild(tdSize);

          var tdAction = document.createElement("td");
          var btn = document.createElement("button");
          btn.className = "cae-btn";
          btn.style.padding = "4px 10px";
          btn.style.fontSize = "13px";
          btn.textContent = "Importar";
          btn.addEventListener("click", function () {
            startImport(f.name, f.path_lower);
          });
          tdAction.appendChild(btn);
          tr.appendChild(tdAction);

          tbody.appendChild(tr);
        });

        tableContainer.hidden = false;
        status.textContent = data.files.length + " ficheiro(s) encontrado(s)";
      } catch (err) {
        loading.hidden = true;
        errorContainer.textContent = err.message;
        errorContainer.hidden = false;
        status.textContent = "Erro";
      }
    }

    async function startImport(name, path) {
      processingName.textContent = name;
      overlay.hidden = false;

      try {
        var resp = await fetch("/saldos-esperados/data/dropbox/process", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ path: path, filename: name }),
        });
        var data = await resp.json();
        if (!resp.ok) throw new Error(data.error || "Erro na importação");

        processingName.textContent = "Concluído!";
        setTimeout(function () {
          window.location.href = "/saldos-esperados/dashboard";
        }, 800);
      } catch (err) {
        overlay.hidden = true;
        window.caeToast("Erro: " + err.message, 5000);
      }
    }

    function formatSize(bytes) {
      if (!bytes) return "0 B";
      var k = 1024;
      var sizes = ["B", "KB", "MB", "GB"];
      var i = Math.floor(Math.log(bytes) / Math.log(k));
      return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
    }
  });
})();
