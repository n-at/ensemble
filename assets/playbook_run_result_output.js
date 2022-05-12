(() => {

    ///////////////////////////////////////////////////////////////////////////
    //collapse empty output blocks

    $(() => {
        $('.card .card-body').each((idx, el) => {
            const $el = $(el);
            if ($el.text().trim().length === 0) {
                $el.addClass('card-body-collapse');
            }
        });
    });

    ///////////////////////////////////////////////////////////////////////////
    //diff

    $(() => {
        $('.diff').each(makeDiff);
    });

    function makeDiff(idx, el) {
        const $el = $(el);

        const beforeHeader = $el.find('.diff-before-header').text();
        const before = $el.find('.diff-before-content').text();

        const afterHeader = $el.find('.diff-after-header').text();
        const after = $el.find('.diff-after-content').text();

        const diff = Diff.createTwoFilesPatch(beforeHeader, afterHeader, before, after);

        const configuration = {
            inputFormat: 'diff',
            matching: 'lines',
            outputFormat: 'side-by-side',
            drawFileList: false,
            showFiles: true,
        };
        new Diff2HtmlUI(el, diff, configuration).draw();
    }

    ///////////////////////////////////////////////////////////////////////////

    $(() => {
        const ansi = new AnsiUp();

        $('.ansi-output').each((idx, el) => {
            const $el = $(el);
            $el.html(ansi.ansi_to_html($el.text()));
        })
    });

})();


