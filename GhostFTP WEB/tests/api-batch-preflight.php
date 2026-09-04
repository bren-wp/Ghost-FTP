<?php
declare(strict_types=1);

$source = file_get_contents(__DIR__ . '/../api.php');
if (!is_string($source)) {
    fwrite(STDERR, "FAIL: unable to inspect API batch-delete source.\n");
    exit(1);
}
$start = strpos($source, "case 'bulk_delete':");
$end = $start !== false ? strpos($source, "case 'copy':", $start) : false;
if ($start === false || $end === false || $end <= $start) {
    fwrite(STDERR, "FAIL: unable to isolate bulk_delete API branch.\n");
    exit(1);
}
$block = substr($source, $start, $end - $start);
$bufferPos = strpos($block, '$validatedItems = [];');
$validateLoopPos = strpos($block, 'foreach ($items as $item)');
$bufferAppendPos = strpos($block, '$validatedItems[] =');
$executeLoopPos = strpos($block, 'foreach ($validatedItems as $item)');
$deletePos = strpos($block, '$ops->deleteRecursive(', $executeLoopPos === false ? 0 : $executeLoopPos);
$malformedPos = strpos($block, 'Popis za skupno brisanje sadrži neispravnu stavku.');
$typePos = strpos($block, 'Vrsta stavke za skupno brisanje nije valjana.');
if (
    $bufferPos === false
    || $validateLoopPos === false
    || $bufferAppendPos === false
    || $executeLoopPos === false
    || $deletePos === false
    || $malformedPos === false
    || $typePos === false
    || !($bufferPos < $validateLoopPos
        && $validateLoopPos < $bufferAppendPos
        && $bufferAppendPos < $executeLoopPos
        && $executeLoopPos < $deletePos
        && $malformedPos < $executeLoopPos
        && $typePos < $executeLoopPos)
) {
    fwrite(STDERR, "FAIL: bulk_delete does not fully validate its batch before remote mutation.\n");
    exit(1);
}

echo "WEB_API_BATCH_PREFLIGHT_TEST=PASS\n";
